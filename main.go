package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fromFile := flag.String("from-file", "", "Load results file from <path>")
	toFile := flag.String("to-file", "", "Save results to <path>")
	deleteDupesIn := flag.String("delete-dupes-in", "", "Delete duplicates if they are contained in <path>")
	force := flag.Bool("force", false, "Actually delete files. Without this options, the files to be deleted are only printed")
	flag.Parse()

	fmt.Printf("fromFile: \"%v\"\n", *fromFile)
	fmt.Printf("toFile: \"%v\"\n", *toFile)
	fmt.Printf("deleteDupesIn: \"%v\"\n", *deleteDupesIn)
	fmt.Printf("force: \"%v\"\n", *force)

	filesMap := newFilesMap()
	if *fromFile != "" {
		fmt.Println("Loading file", *fromFile)

		byteValue, _ := ioutil.ReadFile(*fromFile)
		err := json.Unmarshal(byteValue, &filesMap.FilesBySize)
		if err != nil {
			panic(err)
		}
	} else {
		for _, path := range flag.Args() {
			filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				filesMap.Add(path, info)
				return nil
			})
		}
	}

	if *toFile != "" && *fromFile == "" {
		json, _ := json.MarshalIndent(filesMap.FilesBySize, "", "  ")
		ioutil.WriteFile(*toFile, json, 644)
	}

	if *deleteDupesIn != "" {
		deleteIn := filepath.Clean(*deleteDupesIn)
		for size := range filesMap.FilesBySize {
			for hash := range filesMap.FilesBySize[size] {
				duplicateFiles := filesMap.FilesBySize[size][hash]
				if len(duplicateFiles) <= 1 {
					continue
				}

				fmt.Println("DupeGroup")
				for _, file := range duplicateFiles {
					if strings.HasPrefix(filepath.Clean(file), deleteIn) {
						fmt.Println("d", file)
					} else {
						fmt.Println("k", file)
					}
					if !*force {
					}
				}
				fmt.Println("")
			}
		}
	}
}

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
