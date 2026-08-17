package utils

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/goastian/astiango-hub/core/entity"
	"golang.org/x/sync/singleflight"
)

func OpenFile(fileName string) *os.File {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		logger.Errorf("create file error: %s, file_name: %s", err.Error(), fileName)
		return nil
	}
	return file
}

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// ListDir returns a list of files metadata in the directory
func ListDir(path string) ([]fs.FileInfo, error) {
	list, err := os.ReadDir(path)
	if err != nil {
		logger.Errorf("read dir error: %v, path: %s", err, path)
		return nil, err
	}

	var res []fs.FileInfo
	for _, item := range list {
		info, err := item.Info()
		if err != nil {
			logger.Errorf("get file info error: %v, path: %s", err, item.Name())
			return nil, err
		}
		res = append(res, info)
	}
	return res, nil
}

func ZipDirectory(dir, filePath string) error {
	zipFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	baseDir := filepath.Dir(dir)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

// CopyFile File copies a single file from src to dst
func CopyFile(src, dst string) error {
	var err error
	var srcFd *os.File
	var dstFd *os.File
	var srcInfo os.FileInfo

	if srcFd, err = os.Open(src); err != nil {
		return err
	}
	defer srcFd.Close()

	if dstFd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstFd.Close()

	if _, err = io.Copy(dstFd, srcFd); err != nil {
		return err
	}
	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// CopyDir Dir copies a whole directory recursively
func CopyDir(src string, dst string) error {
	var err error
	var fds []os.DirEntry
	var srcInfo os.FileInfo

	if srcInfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	if fds, err = os.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = CopyDir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = CopyFile(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}

func GetFileHash(filePath string) (res string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

const IgnoreFileRegexPattern = `(^\.git|^node_modules|__pycache__)/|\.(tmp|temp|log|swp|swo|bak|orig|lock|pid|pyc|pyo)$`
const scanDirectoryCacheTTL = 3 * time.Second

var (
	scanDirectoryGroup singleflight.Group
	scanDirectoryCache = struct {
		sync.RWMutex
		items map[string]scanDirectoryCacheEntry
	}{items: make(map[string]scanDirectoryCacheEntry)}
)

type scanDirectoryCacheEntry struct {
	data      entity.FsFileInfoMap
	expiresAt time.Time
}

func ScanDirectory(dir string) (entity.FsFileInfoMap, error) {
	if res, ok := getScanDirectoryCache(dir); ok {
		return cloneFsFileInfoMap(res), nil
	}

	v, err, _ := scanDirectoryGroup.Do(dir, func() (any, error) {
		if res, ok := getScanDirectoryCache(dir); ok {
			return cloneFsFileInfoMap(res), nil
		}

		files, err := scanDirectoryInternal(dir)
		if err != nil {
			return nil, err
		}

		setScanDirectoryCache(dir, files)
		return cloneFsFileInfoMap(files), nil
	})
	if err != nil {
		return nil, err
	}

	res, ok := v.(entity.FsFileInfoMap)
	if !ok {
		return nil, fmt.Errorf("unexpected cache value type: %T", v)
	}

	return cloneFsFileInfoMap(res), nil
}

func scanDirectoryInternal(dir string) (entity.FsFileInfoMap, error) {
	files := make(entity.FsFileInfoMap)

	ignoreRegex, err := regexp.Compile(IgnoreFileRegexPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile ignore pattern: %v", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if ignoreRegex.MatchString(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		var hash string
		if !info.IsDir() {
			hash, err = GetFileHash(path)
			if err != nil {
				return err
			}
		}

		files[relPath] = entity.FsFileInfo{
			Name:      info.Name(),
			Path:      relPath,
			FullPath:  path,
			Extension: filepath.Ext(path),
			IsDir:     info.IsDir(),
			FileSize:  info.Size(),
			ModTime:   info.ModTime(),
			Mode:      info.Mode(),
			Hash:      hash,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func getScanDirectoryCache(dir string) (entity.FsFileInfoMap, bool) {
	scanDirectoryCache.RLock()
	defer scanDirectoryCache.RUnlock()

	entry, ok := scanDirectoryCache.items[dir]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func setScanDirectoryCache(dir string, data entity.FsFileInfoMap) {
	scanDirectoryCache.Lock()
	defer scanDirectoryCache.Unlock()

	scanDirectoryCache.items[dir] = scanDirectoryCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(scanDirectoryCacheTTL),
	}
}

func cloneFsFileInfoMap(src entity.FsFileInfoMap) entity.FsFileInfoMap {
	if src == nil {
		return nil
	}
	dst := make(entity.FsFileInfoMap, len(src))
	maps.Copy(dst, src)
	return dst
}
