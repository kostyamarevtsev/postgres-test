package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func initBackup(target, backup string) error {
	err := os.RemoveAll(backup)
	if err != nil {
		return err
	}

	err = os.MkdirAll(backup, 0755)
	if err != nil {
		return err
	}

	fileInfos, err := ioutil.ReadDir(target)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileInfos {
		srcPath := filepath.Join(target, fileInfo.Name())
		destPath := filepath.Join(backup, fileInfo.Name())

		if fileInfo.IsDir() {
			err = initBackup(srcPath, destPath)
			if err != nil {
				return err
			}
		} else {
			err = cp(srcPath, destPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func cp(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
