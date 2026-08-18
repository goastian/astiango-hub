package controllers

import (
	"context"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/goastian/astiango-hub/core/entity"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/juju/errors"
	"golang.org/x/sync/semaphore"
)

var (
	syncDownloadSemaphore = semaphore.NewWeighted(utils.GetSyncDownloadMaxConcurrency())
	syncDownloadInFlight  int64
	syncScanSemaphore     = semaphore.NewWeighted(10) // Limit concurrent scan requests
	syncScanInFlight      int64
)

func GetSyncScan(c *gin.Context) (response *Response[entity.FsFileInfoMap], err error) {
	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Rate limiting for scan requests
	if err := syncScanSemaphore.Acquire(ctx, 1); err != nil {
		logger.Warnf("failed to acquire sync scan slot for id=%s path=%s: %v", c.Param("id"), c.Param("path"), err)
		return GetErrorResponse[entity.FsFileInfoMap](errors.Annotate(err, "server overloaded, please retry"))
	}
	current := atomic.AddInt64(&syncScanInFlight, 1)
	logger.Debugf("sync scan in-flight=%d id=%s path=%s", current, c.Param("id"), c.Param("path"))
	defer func() {
		newVal := atomic.AddInt64(&syncScanInFlight, -1)
		logger.Debugf("sync scan completed in-flight=%d id=%s path=%s", newVal, c.Param("id"), c.Param("path"))
		syncScanSemaphore.Release(1)
	}()

	syncRoot, err := utils.ResolvePathWithinRoot(utils.GetWorkspace(), c.Param("id"))
	if err != nil {
		return GetErrorResponse[entity.FsFileInfoMap](err)
	}
	dirPath, err := utils.ResolvePathWithinRoot(syncRoot, c.Param("path"))
	if err != nil {
		return GetErrorResponse[entity.FsFileInfoMap](err)
	}
	files, err := utils.ScanDirectory(dirPath)
	if err != nil {
		logger.Warnf("sync scan failed id=%s path=%s: %v", c.Param("id"), c.Param("path"), err)
		return GetErrorResponse[entity.FsFileInfoMap](err)
	}
	return GetDataResponse(files)
}

func GetSyncDownload(c *gin.Context) (err error) {
	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	if err := syncDownloadSemaphore.Acquire(ctx, 1); err != nil {
		logger.Warnf("failed to acquire sync download slot for id=%s path=%s: %v", c.Param("id"), c.Query("path"), err)
		return errors.Annotate(err, "acquire sync download slot")
	}
	current := atomic.AddInt64(&syncDownloadInFlight, 1)
	logger.Debugf("sync download in-flight=%d id=%s path=%s", current, c.Param("id"), c.Query("path"))
	defer func() {
		newVal := atomic.AddInt64(&syncDownloadInFlight, -1)
		logger.Debugf("sync download completed in-flight=%d id=%s path=%s", newVal, c.Param("id"), c.Query("path"))
		syncDownloadSemaphore.Release(1)
	}()

	syncRoot, err := utils.ResolvePathWithinRoot(utils.GetWorkspace(), c.Param("id"))
	if err != nil {
		return err
	}
	filePath, err := utils.ResolvePathWithinRoot(syncRoot, c.Query("path"))
	if err != nil {
		return err
	}
	if !utils.Exists(filePath) {
		return errors.NotFoundf("file not exists: %s", filePath)
	}
	c.File(filePath)
	return nil
}
