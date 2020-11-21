package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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

	json, _ := json.MarshalIndent(filesMap.FilesBySize, "", "  ")
	fmt.Printf("\n\n\n%s\n\n\n", json)
}

type filesMap struct {
	FilesBySize map[int64]map[string][]*fileEntry
}

func (fm *filesMap) Add(path string, info os.FileInfo) error {
	if info.IsDir() {
		return nil
	}

	fileInfo := &fileEntry{
		Path: path,
		Size: info.Size(),
	}

	filesByHash := fm.FilesBySize[fileInfo.Size]

	// first file with same size
	// => create new map for size
	if filesByHash == nil {
		filesByHash = map[string][]*fileEntry{}
		fm.FilesBySize[fileInfo.Size] = filesByHash
		filesByHash[""] = []*fileEntry{fileInfo}
		return nil
	}

	// second file with same size
	// => calculate hashes for all entries
	if _, hasEmptyHash := filesByHash[""]; hasEmptyHash {
		err := appendByFileHash(filesByHash, fileInfo)
		err2 := appendByFileHash(filesByHash, filesByHash[""][0])

		delete(filesByHash, "")

		if err != nil {
			return err
		}

		return err2
	}

	return appendByFileHash(filesByHash, fileInfo)
}

func appendByFileHash(filesByHash map[string][]*fileEntry, fileInfo *fileEntry) error {
	hash, err := calculateHash(fileInfo.Path)

	if err != nil {
		return err
	}

	if _, ok := filesByHash[hash]; ok {
		filesByHash[hash] = append(filesByHash[hash], fileInfo)
	} else {
		filesByHash[hash] = []*fileEntry{fileInfo}
	}
	return nil
}

func newFilesMap() filesMap {
	return filesMap{
		FilesBySize: map[int64]map[string][]*fileEntry{},
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
