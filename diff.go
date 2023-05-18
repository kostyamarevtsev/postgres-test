package main

import (
	"io/ioutil"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var dmp = diffmatchpatch.New()

func diff(file1Path, file2Path string) ([]diffmatchpatch.Diff, error) {
	text1, err := readFileContents(file1Path)
	if err != nil {
		return nil, err
	}

	text2, err := readFileContents(file2Path)
	if err != nil {
		return nil, err
	}

	return dmp.DiffMain(text1, text2, false), nil
}

func readFileContents(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
