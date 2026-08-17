package utils

import (
	"errors"
	"github.com/goastian/astiango-hub/core/models/models"
	"path/filepath"
)

func GetSpiderRootPath(s *models.Spider) (rootPath string, err error) {
	// check git permission
	if !IsPro() && !s.GitId.IsZero() {
		return "", errors.New("git is not allowed in the community version")
	}

	// if git id is zero, return spider id as root path
	if s.GitId.IsZero() {
		return s.Id.Hex(), nil
	}

	return filepath.Join(s.GitId.Hex(), s.GitRootPath), nil
}

func GetSpiderFullRootPath(s *models.Spider) (rootPath string, err error) {
	// workspace path
	workspacePath := GetWorkspace()

	// get spider root path
	rootPath, err = GetSpiderRootPath(s)
	if err != nil {
		return "", err
	}

	return filepath.Join(workspacePath, rootPath), nil
}
