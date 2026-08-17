package entity

import (
	"os"
	"time"
)

type FsFileInfo struct {
	Name      string        `json:"name"`      // file name
	Path      string        `json:"path"`      // file path
	FullPath  string        `json:"full_path"` // file full path
	Extension string        `json:"extension"` // file extension
	IsDir     bool          `json:"is_dir"`    // whether it is directory
	FileSize  int64         `json:"file_size"` // file size (bytes)
	Children  []*FsFileInfo `json:"children"`  // children for subdirectory
	ModTime   time.Time     `json:"mod_time"`  // modification time
	Mode      os.FileMode   `json:"mode"`      // file mode
	Hash      string        `json:"hash"`      // file hash
}

type FsFileInfoMap map[string]FsFileInfo
