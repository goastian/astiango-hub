package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/utils"
	grpc2 "github.com/goastian/astiango-hub/grpc"
)

type SyncServiceServer struct {
	grpc2.UnimplementedSyncServiceServer
	*utils.Logger

	// Request deduplication: key = spider_id:path
	activeScans   map[string]*activeScanState
	activeScansMu sync.RWMutex

	// Cache: avoid rescanning within TTL
	scanCache    map[string]*cachedScanResult
	scanCacheMu  sync.RWMutex
	scanCacheTTL time.Duration
	chunkSize    int
}

type activeScanState struct {
	inProgress  bool
	waitChan    chan *cachedScanResult // Broadcast to waiting requests
	subscribers int
}

type cachedScanResult struct {
	files     []*grpc2.FileInfo
	timestamp time.Time
	err       error
}

func NewSyncServiceServer() *SyncServiceServer {
	return &SyncServiceServer{
		Logger:       utils.NewLogger("SyncServiceServer"),
		activeScans:  make(map[string]*activeScanState),
		scanCache:    make(map[string]*cachedScanResult),
		scanCacheTTL: 60 * time.Second, // Longer TTL for streaming
		chunkSize:    100,              // Files per chunk
	}
}

// StreamFileScan streams file information to worker
func (s *SyncServiceServer) StreamFileScan(
	req *grpc2.FileSyncRequest,
	stream grpc2.SyncService_StreamFileScanServer,
) error {
	cacheKey := req.SpiderId + ":" + req.Path

	s.Debugf("file scan request from node %s for spider %s, path %s", req.NodeKey, req.SpiderId, req.Path)

	// Check cache first
	if result := s.getCachedScan(cacheKey); result != nil {
		s.Debugf("returning cached scan for %s", cacheKey)
		return s.streamCachedResult(stream, result)
	}

	// Deduplicate concurrent requests
	result, err := s.getOrWaitForScan(cacheKey, func() (*cachedScanResult, error) {
		return s.performScan(req)
	})

	if err != nil {
		s.Errorf("scan failed for %s: %v", cacheKey, err)
		return stream.Send(&grpc2.FileScanChunk{
			IsComplete: true,
			Error:      err.Error(),
		})
	}

	return s.streamCachedResult(stream, result)
}

// performScan does the actual directory scan
func (s *SyncServiceServer) performScan(req *grpc2.FileSyncRequest) (*cachedScanResult, error) {
	workspacePath := utils.GetWorkspace()
	dirPath := filepath.Join(workspacePath, req.SpiderId, req.Path)

	s.Infof("performing directory scan for %s", dirPath)

	// Use existing ScanDirectory which has singleflight and short-term cache
	fileMap, err := utils.ScanDirectory(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	// Convert to protobuf format
	files := make([]*grpc2.FileInfo, 0, len(fileMap))
	for _, f := range fileMap {
		files = append(files, &grpc2.FileInfo{
			Name:      f.Name,
			Path:      f.Path,
			FullPath:  f.FullPath,
			Extension: f.Extension,
			IsDir:     f.IsDir,
			FileSize:  f.FileSize,
			ModTime:   f.ModTime.Unix(),
			Mode:      uint32(f.Mode),
			Hash:      f.Hash,
		})
	}

	result := &cachedScanResult{
		files:     files,
		timestamp: time.Now(),
	}

	// Cache the result
	cacheKey := req.SpiderId + ":" + req.Path
	s.scanCacheMu.Lock()
	s.scanCache[cacheKey] = result
	s.scanCacheMu.Unlock()

	s.Infof("scanned %d files from %s", len(files), dirPath)

	return result, nil
}

// streamCachedResult streams the cached result in chunks
func (s *SyncServiceServer) streamCachedResult(
	stream grpc2.SyncService_StreamFileScanServer,
	result *cachedScanResult,
) error {
	totalFiles := len(result.files)

	for i := 0; i < totalFiles; i += s.chunkSize {
		end := i + s.chunkSize
		if end > totalFiles {
			end = totalFiles
		}

		chunk := &grpc2.FileScanChunk{
			Files:      result.files[i:end],
			IsComplete: end >= totalFiles,
			TotalFiles: int32(totalFiles),
		}

		if err := stream.Send(chunk); err != nil {
			return fmt.Errorf("failed to send chunk: %w", err)
		}
	}

	return nil
}

// getOrWaitForScan implements request deduplication
func (s *SyncServiceServer) getOrWaitForScan(
	key string,
	scanFunc func() (*cachedScanResult, error),
) (*cachedScanResult, error) {
	s.activeScansMu.Lock()

	state, exists := s.activeScans[key]
	if exists && state.inProgress {
		// Another request is already scanning, wait for it
		state.subscribers++
		waitChan := state.waitChan
		s.activeScansMu.Unlock()

		s.Debugf("waiting for ongoing scan: %s", key)
		result := <-waitChan
		return result, result.err
	}

	// We're the first request, start scanning
	state = &activeScanState{
		inProgress:  true,
		waitChan:    make(chan *cachedScanResult, 10),
		subscribers: 0,
	}
	s.activeScans[key] = state
	s.activeScansMu.Unlock()

	s.Debugf("initiating new scan: %s", key)

	// Perform scan
	result, err := scanFunc()
	if err != nil {
		result = &cachedScanResult{err: err}
	}

	// Broadcast to waiting requests
	s.activeScansMu.Lock()
	for i := 0; i < state.subscribers; i++ {
		state.waitChan <- result
	}
	delete(s.activeScans, key)
	close(state.waitChan)
	s.activeScansMu.Unlock()

	s.Debugf("scan complete for %s, notified %d subscribers", key, state.subscribers)

	return result, err
}

func (s *SyncServiceServer) getCachedScan(key string) *cachedScanResult {
	s.scanCacheMu.RLock()
	defer s.scanCacheMu.RUnlock()

	result, exists := s.scanCache[key]
	if !exists {
		return nil
	}

	// Check if cache expired
	if time.Since(result.timestamp) > s.scanCacheTTL {
		return nil
	}

	return result
}

// StreamFileDownload streams file content to worker
func (s *SyncServiceServer) StreamFileDownload(
	req *grpc2.FileDownloadRequest,
	stream grpc2.SyncService_StreamFileDownloadServer,
) error {
	workspacePath := utils.GetWorkspace()
	filePath := filepath.Join(workspacePath, req.SpiderId, req.Path)

	s.Infof("streaming file download: %s", filePath)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return stream.Send(&grpc2.FileDownloadChunk{
			IsComplete: true,
			Error:      fmt.Sprintf("failed to open file: %v", err),
		})
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return stream.Send(&grpc2.FileDownloadChunk{
			IsComplete: true,
			Error:      fmt.Sprintf("failed to stat file: %v", err),
		})
	}

	// Stream file in chunks
	const bufferSize = 64 * 1024 // 64KB chunks
	buffer := make([]byte, bufferSize)
	totalBytes := fileInfo.Size()
	bytesSent := int64(0)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunk := &grpc2.FileDownloadChunk{
				Data:       buffer[:n],
				IsComplete: false,
				TotalBytes: totalBytes,
			}

			if err := stream.Send(chunk); err != nil {
				return fmt.Errorf("failed to send chunk: %w", err)
			}

			bytesSent += int64(n)
		}

		if err != nil {
			if err.Error() == "EOF" {
				// Send final chunk
				return stream.Send(&grpc2.FileDownloadChunk{
					IsComplete: true,
					TotalBytes: totalBytes,
				})
			}
			return stream.Send(&grpc2.FileDownloadChunk{
				IsComplete: true,
				Error:      fmt.Sprintf("read error: %v", err),
			})
		}
	}
}

var _syncServiceServer *SyncServiceServer
var _syncServiceServerOnce sync.Once

func GetSyncServiceServer() *SyncServiceServer {
	_syncServiceServerOnce.Do(func() {
		_syncServiceServer = NewSyncServiceServer()
	})
	return _syncServiceServer
}
