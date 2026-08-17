package fs

import (
	"github.com/goastian/astiango-hub/core/utils"
	"path/filepath"
)

func GetBaseFileFsSvc(rootPath string) (svc *Service, err error) {
	workspacePath := utils.GetWorkspace()
	fsSvc := NewFsService(filepath.Join(workspacePath, rootPath))

	return fsSvc, nil
}
