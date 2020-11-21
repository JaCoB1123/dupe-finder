package main

import "os"

// FilesMap is a struct for listing files by Size and Hash to search for duplicates
type FilesMap struct {
	FilesBySize map[int64]map[string][]string
}

// Add a file to the Map and calculate hash on demand
func (fm *FilesMap) Add(path string, info os.FileInfo) error {
	if info.IsDir() {
		return nil
	}

	fileInfo := path

	filesByHash := fm.FilesBySize[info.Size()]

	// first file with same size
	// => create new map for size
	if filesByHash == nil {
		filesByHash = map[string][]string{}
		fm.FilesBySize[info.Size()] = filesByHash
		filesByHash[""] = []string{fileInfo}
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

	// for later files always append by hash
	return appendByFileHash(filesByHash, fileInfo)
}

func appendByFileHash(filesByHash map[string][]string, fileInfo string) error {
	hash, err := calculateHash(fileInfo)

	if err != nil {
		return err
	}

	if _, ok := filesByHash[hash]; ok {
		filesByHash[hash] = append(filesByHash[hash], fileInfo)
	} else {
		filesByHash[hash] = []string{fileInfo}
	}
	return nil
}

func newFilesMap() *FilesMap {
	return &FilesMap{
		FilesBySize: map[int64]map[string][]string{},
	}
}
