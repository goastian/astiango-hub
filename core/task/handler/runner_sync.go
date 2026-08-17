package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/utils"
)

const (
	syncHTTPRequestMaxRetries     = 5
	syncHTTPRequestInitialBackoff = 200 * time.Millisecond
	syncHTTPRequestMaxBackoff     = 3 * time.Second
	syncHTTPRequestClientTimeout  = 30 * time.Second
)

var (
	syncHttpClient = &http.Client{Timeout: syncHTTPRequestClientTimeout}
	jitterRand     = rand.New(rand.NewSource(time.Now().UnixNano()))
	jitterMutex    sync.Mutex
)

// syncFiles synchronizes files between master and worker nodes.
// It switches between gRPC streaming and HTTP based on the feature flag.
func (r *Runner) syncFiles() (err error) {
	if utils.IsSyncGrpcEnabled() {
		return r.syncFilesGRPC()
	}
	return r.syncFilesHTTP()
}

// syncFilesHTTP synchronizes files using HTTP/JSON (legacy implementation):
// 1. Gets file list from master
// 2. Compares with local files
// 3. Downloads new/modified files
// 4. Deletes files that no longer exist on master
func (r *Runner) syncFilesHTTP() (err error) {
	r.Infof("starting HTTP file synchronization for spider: %s", r.s.Id.Hex())

	workingDir := ""
	if !r.s.GitId.IsZero() {
		workingDir = r.s.GitRootPath
		r.Debugf("using git root path: %s", workingDir)
	}

	// get file list from master
	r.Infof("fetching file list from master node")
	params := url.Values{
		"path": []string{workingDir},
	}
	resp, err := r.performHttpRequest("GET", "/scan", params)
	if err != nil {
		r.Errorf("error getting file list from master: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Validate Content-Type to detect non-JSON responses early
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		r.Errorf("unexpected Content-Type: %s (expected application/json)", contentType)
		r.Errorf("URL: %s, Status: %d", resp.Request.URL.String(), resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 500 {
			r.Errorf("Response preview: %s...", string(body[:500]))
		} else {
			r.Errorf("Response body: %s", string(body))
		}
		return fmt.Errorf("master returned non-JSON response (Content-Type: %s, Status: %d)", contentType, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		r.Errorf("error reading response body: %v", err)
		return err
	}
	var response struct {
		Data entity.FsFileInfoMap `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		r.Errorf("error unmarshaling JSON for URL: %s", resp.Request.URL.String())
		r.Errorf("error details: %v", err)
		r.Errorf("response body length: %d bytes", len(body))
		if len(body) > 500 {
			r.Errorf("response preview: %s...", string(body[:500]))
		} else if len(body) > 0 {
			r.Errorf("response body: %s", string(body))
		}
		return err
	}

	// create a map for master files
	masterFilesMap := make(entity.FsFileInfoMap)
	for _, file := range response.Data {
		masterFilesMap[file.Path] = file
	}

	// create working directory if not exists
	if _, err := os.Stat(r.cwd); os.IsNotExist(err) {
		if err := os.MkdirAll(r.cwd, os.ModePerm); err != nil {
			r.Errorf("error creating worker directory: %v", err)
			return err
		}
	}

	// get file list from worker
	workerFiles, err := utils.ScanDirectory(r.cwd)
	if err != nil {
		r.Errorf("error scanning worker directory: %v", err)
		return err
	}

	// delete files that are deleted on master node
	for path, workerFile := range workerFiles {
		if _, exists := masterFilesMap[path]; !exists {
			r.Infof("deleting file: %s", path)
			err := os.Remove(workerFile.FullPath)
			if err != nil {
				r.Errorf("error deleting file: %v", err)
			}
		}
	}

	// set up wait group and error channel
	var wg sync.WaitGroup
	pool := make(chan struct{}, 10)

	// download files that are new or modified on master node
	for path, masterFile := range masterFilesMap {
		workerFile, exists := workerFiles[path]
		if !exists || masterFile.Hash != workerFile.Hash {
			wg.Add(1)

			// acquire token
			pool <- struct{}{}

			// start goroutine to synchronize file or directory
			go func(path string, masterFile *entity.FsFileInfo) {
				defer wg.Done()

				if masterFile.IsDir {
					r.Infof("directory needs to be synchronized: %s", path)
					_err := os.MkdirAll(filepath.Join(r.cwd, path), masterFile.Mode)
					if _err != nil {
						r.Errorf("error creating directory: %v", _err)
						err = errors.Join(err, _err)
					}
				} else {
					r.Infof("file needs to be synchronized: %s", path)
					_err := r.downloadFile(path, filepath.Join(r.cwd, path), masterFile)
					if _err != nil {
						r.Errorf("error downloading file: %v", _err)
						err = errors.Join(err, _err)
					}
				}

				// release token
				<-pool

			}(path, &masterFile)
		}
	}

	// wait for all files and directories to be synchronized
	wg.Wait()

	r.Infof("file synchronization completed successfully")
	return err
}

func (r *Runner) performHttpRequest(method, path string, params url.Values) (*http.Response, error) {
	path = strings.TrimPrefix(path, "/")

	var id string
	if r.s.GitId.IsZero() {
		id = r.s.Id.Hex()
	} else {
		id = r.s.GitId.Hex()
	}
	requestUrl := fmt.Sprintf("%s/sync/%s/%s?%s", utils.GetApiEndpoint(), id, path, params.Encode())

	backoff := syncHTTPRequestInitialBackoff
	var lastErr error

	for attempt := range syncHTTPRequestMaxRetries {
		req, err := http.NewRequest(method, requestUrl, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		for k, v := range r.getHttpRequestHeaders() {
			req.Header.Set(k, v)
		}

		resp, err := syncHttpClient.Do(req)
		if err == nil && !shouldRetryStatus(resp.StatusCode) {
			return resp, nil
		}

		if err == nil {
			lastErr = fmt.Errorf("received retryable status %d for %s", resp.StatusCode, requestUrl)
			_ = resp.Body.Close()
		} else {
			lastErr = err
		}

		wait := backoff + jitterDuration(backoff/2)
		if wait > syncHTTPRequestMaxBackoff {
			wait = syncHTTPRequestMaxBackoff
		}

		r.Warnf("retrying %s %s in %s (attempt %d/%d): %v", method, requestUrl, wait, attempt+1, syncHTTPRequestMaxRetries, lastErr)
		time.Sleep(wait)

		if backoff < syncHTTPRequestMaxBackoff {
			backoff *= 2
			if backoff > syncHTTPRequestMaxBackoff {
				backoff = syncHTTPRequestMaxBackoff
			}
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("exceeded max retries for %s", requestUrl)
	}
	return nil, lastErr
}

func shouldRetryStatus(status int) bool {
	if status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || status == http.StatusTooEarly {
		return true
	}
	switch status {
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return status >= 500
}

func jitterDuration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	jitterMutex.Lock()
	defer jitterMutex.Unlock()
	return time.Duration(jitterRand.Int63n(int64(max)))
}

// downloadFile downloads a file from the master node and saves it to the local file system
func (r *Runner) downloadFile(path string, filePath string, fileInfo *entity.FsFileInfo) error {
	r.Debugf("downloading file: %s -> %s", path, filePath)
	params := url.Values{
		"path": []string{path},
	}
	resp, err := r.performHttpRequest("GET", "/download", params)
	if err != nil {
		r.Errorf("error getting file response: %v", err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		r.Errorf("error downloading file: %s", resp.Status)
		return errors.New(resp.Status)
	}
	defer resp.Body.Close()

	// create directory if not exists
	dirPath := filepath.Dir(filePath)
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		r.Errorf("error creating directory: %v", err)
		return err
	}

	// create local file
	out, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileInfo.Mode)
	if err != nil {
		r.Errorf("error creating file: %v", err)
		return err
	}
	defer out.Close()

	// copy file content to local file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		r.Errorf("error copying file: %v", err)
		return err
	}

	r.Debugf("successfully downloaded file: %s (size: %d bytes)", path, fileInfo.FileSize)
	return nil
}

// getHttpRequestHeaders returns the headers for HTTP requests to the master node
func (r *Runner) getHttpRequestHeaders() (headers map[string]string) {
	headers = make(map[string]string)
	headers["Authorization"] = utils.GetAuthKey()
	return headers
}
