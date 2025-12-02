package flags

import (
	"os"
	"path/filepath"
)

// PathFlag accepts path to some file or directory and splits it
// into dirPath and filename (both can be empty)
type PathFlag struct {
	originalPath string
	dirPath      string
	filename     string
}

func (f *PathFlag) Set(path string) error {
	f.originalPath = path

	fileInfo, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if fileInfo != nil && fileInfo.IsDir() {
		f.dirPath = path
		return nil
	}

	f.dirPath, f.filename = filepath.Split(path)
	return nil
}

func (f *PathFlag) String() string {
	return f.originalPath
}

func (f *PathFlag) Clone() any {
	return &PathFlag{
		originalPath: f.originalPath,
		dirPath:      f.dirPath,
		filename:     f.filename,
	}
}

func (f *PathFlag) Dir() string {
	return f.dirPath
}

func (f *PathFlag) Filename() string {
	return f.filename
}
