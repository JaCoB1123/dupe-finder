package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	filesMap := newFilesMap()
	for _, path := range os.Args[1:] {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			filesMap.Add(path, info)
			return nil
		})
		fmt.Println(path)
	}

}

type filesMap struct {
	Files map[int64]map[string]*fileEntry
}

func (fm *filesMap) Add(path string, info os.FileInfo) {
	fileInfo := &fileEntry{
		Path: path,
		Size: info.Size(),
	}

	existing := fm.Files[fileInfo.Size]
	if existing == nil {
		existing = map[string]*fileEntry{}
		fm.Files[fileInfo.Size] = existing
		existing[""] = fileInfo
		return
	}

	fmt.Println("Dupes: " + path)
	fmt.Println(existing[""])
	fm.Files[fileInfo.Size] = nil

}

func newFilesMap() filesMap {
	return filesMap{
		Files: map[int64]map[string]*fileEntry{},
	}
}

type fileEntry struct {
	Path string
	Size int64
}

func calculateHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(h.Sum(nil)), nil
}
