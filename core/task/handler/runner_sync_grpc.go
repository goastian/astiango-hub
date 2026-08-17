package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/goastian/astiango-hub/core/entity"
	client2 "github.com/goastian/astiango-hub/core/grpc/client"
	"github.com/goastian/astiango-hub/core/utils"
	grpc2 "github.com/goastian/astiango-hub/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// syncFilesGRPC replaces HTTP-based syncFiles() with gRPC streaming
func (r *Runner) syncFilesGRPC() (err error) {
	r.Infof("starting gRPC file synchronization for spider: %s", r.s.Id.Hex())

	workingDir := ""
	if !r.s.GitId.IsZero() {
		workingDir = r.s.GitRootPath
		r.Debugf("using git root path: %s", workingDir)
	}

	// Get sync service client
	syncClient, err := client2.GetGrpcClient().GetSyncClient()
	if err != nil {
		r.Errorf("failed to get sync client: %v", err)
		return err
	}

	// Determine the ID to use for sync request
	// For git spiders, use GitId (where files are stored)
	// For regular spiders, use SpiderId
	var syncId string
	if r.s.GitId.IsZero() {
		syncId = r.s.Id.Hex()
	} else {
		syncId = r.s.GitId.Hex()
	}

	// Prepare request
	req := &grpc2.FileSyncRequest{
		SpiderId: syncId,
		Path:     workingDir,
		NodeKey:  utils.GetNodeKey(),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Stream file list from master
	r.Infof("fetching file list from master via gRPC")
	stream, err := syncClient.StreamFileScan(ctx, req)
	if err != nil {
		r.Errorf("failed to start file scan stream: %v", err)
		return err
	}

	// Receive file list in chunks
	masterFilesMap := make(entity.FsFileInfoMap)
	totalFiles := 0

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			// Check for gRPC-specific errors
			if st, ok := status.FromError(err); ok {
				if st.Code() == codes.Unavailable {
					r.Errorf("gRPC service unavailable, will retry: %v", err)
					return fmt.Errorf("gRPC service unavailable: %w", err)
				}
			}
			r.Errorf("error receiving file scan chunk: %v", err)
			return err
		}

		// Check for error in chunk
		if chunk.Error != "" {
			r.Errorf("server error during file scan: %s", chunk.Error)
			return fmt.Errorf("server error: %s", chunk.Error)
		}

		// Process files in chunk
		for _, fileInfo := range chunk.Files {
			fsFileInfo := entity.FsFileInfo{
				Name:      fileInfo.Name,
				Path:      fileInfo.Path,
				FullPath:  fileInfo.FullPath,
				Extension: fileInfo.Extension,
				IsDir:     fileInfo.IsDir,
				FileSize:  fileInfo.FileSize,
				ModTime:   time.Unix(fileInfo.ModTime, 0),
				Mode:      os.FileMode(fileInfo.Mode),
				Hash:      fileInfo.Hash,
			}
			masterFilesMap[fileInfo.Path] = fsFileInfo
		}

		if chunk.IsComplete {
			totalFiles = int(chunk.TotalFiles)
			r.Infof("received complete file list: %d files", totalFiles)
			break
		}
	}

	// Create working directory if not exists
	if _, err := os.Stat(r.cwd); os.IsNotExist(err) {
		if err := os.MkdirAll(r.cwd, os.ModePerm); err != nil {
			r.Errorf("error creating worker directory: %v", err)
			return err
		}
	}

	// Get file list from worker
	workerFiles, err := utils.ScanDirectory(r.cwd)
	if err != nil {
		r.Errorf("error scanning worker directory: %v", err)
		return err
	}

	// Delete files that are deleted on master node
	for path, workerFile := range workerFiles {
		if _, exists := masterFilesMap[path]; !exists {
			r.Infof("deleting file: %s", path)
			err := os.Remove(workerFile.FullPath)
			if err != nil {
				r.Errorf("error deleting file: %v", err)
				return err
			}
		}
	}

	// Download new or modified files
	downloadCount := 0
	for path, masterFile := range masterFilesMap {
		// Skip directories
		if masterFile.IsDir {
			continue
		}

		workerFile, exists := workerFiles[path]
		needsDownload := false

		if !exists {
			r.Debugf("file not found locally: %s", path)
			needsDownload = true
		} else if workerFile.Hash != masterFile.Hash {
			r.Debugf("file hash mismatch: %s (local: %s, master: %s)", path, workerFile.Hash, masterFile.Hash)
			needsDownload = true
		}

		if needsDownload {
			// For git spiders with sub-folders, prepend GitRootPath to the download path
			// because the server stores files at workspace/{git_id}/{git_root_path}/...
			downloadPath := path
			if workingDir != "" {
				downloadPath = filepath.Join(workingDir, path)
			}
			if err := r.downloadFileGRPC(syncClient, syncId, downloadPath); err != nil {
				r.Errorf("error downloading file %s: %v", path, err)
				return err
			}
			downloadCount++
		}
	}

	r.Infof("file synchronization complete: %d files downloaded", downloadCount)
	return nil
}

// downloadFileGRPC downloads a single file from master via gRPC streaming
func (r *Runner) downloadFileGRPC(client grpc2.SyncServiceClient, spiderId, path string) error {
	r.Debugf("downloading file via gRPC: %s", path)

	// Prepare request
	req := &grpc2.FileDownloadRequest{
		SpiderId: spiderId,
		Path:     path,
		NodeKey:  utils.GetNodeKey(),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Stream file download
	stream, err := client.StreamFileDownload(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start download stream: %w", err)
	}

	// Create target file
	targetPath := fmt.Sprintf("%s/%s", r.cwd, path)

	// Create directory if not exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Receive file content in chunks
	bytesReceived := int64(0)
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error receiving download chunk: %w", err)
		}

		// Check for error in chunk
		if chunk.Error != "" {
			return fmt.Errorf("server error during download: %s", chunk.Error)
		}

		// Write chunk data
		if len(chunk.Data) > 0 {
			n, err := file.Write(chunk.Data)
			if err != nil {
				return fmt.Errorf("error writing file: %w", err)
			}
			bytesReceived += int64(n)
		}

		if chunk.IsComplete {
			r.Debugf("download complete: %s (%d bytes)", path, bytesReceived)
			break
		}
	}

	return nil
}
