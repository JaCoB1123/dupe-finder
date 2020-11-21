package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var fromFile = flag.String("from-file", "", "Load results file from <path>")
var toFile = flag.String("to-file", "", "Save results to <path>")
var deleteDupesIn = flag.String("delete-dupes-in", "", "Delete duplicates if they are contained in <path>")
var promptForDelete = flag.Bool("delete-prompt", false, "Ask which file to keep for each dupe-set")
var moveToFolder = flag.String("move-files", "", "Move files to <path> instead of deleting them")
var force = flag.Bool("force", false, "Actually delete files. Without this options, the files to be deleted are only printed")
var verbose = flag.Bool("verbose", false, "Output additional information")

func delete(path string) {
	if !*force {
		return
	}

	if *moveToFolder == "" {
		os.Remove(path)
		return
	}

	moveButDontOvewrite(path, *moveToFolder)
}

func moveButDontOvewrite(path string, targetPath string) {
	num := 0

	filename := filepath.Base(path)

	target := filepath.Join(targetPath, filename)

	for {
		_, err := os.Stat(target)
		if os.IsNotExist(err) {
			os.Rename(path, target)
			return
		}

		target = filepath.Join(targetPath, filename+"."+strconv.Itoa(num))
		num++
	}
}

func main() {
	flag.Parse()

	if *verbose {
		printConfiguration()
	}

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

				for _, file := range duplicateFiles {
					if strings.HasPrefix(filepath.Clean(file), deleteIn) {
						fmt.Println("Would delete ", file)
						if *force {
							delete(file)
						}
					}
				}
			}
		}
	} else if *promptForDelete {
		reader := bufio.NewReader(os.Stdin)
		for size := range filesMap.FilesBySize {
			for hash := range filesMap.FilesBySize[size] {
				duplicateFiles := filesMap.FilesBySize[size][hash]
				if len(duplicateFiles) <= 1 {
					continue
				}

				fmt.Print("\033[H\033[2J")
				for i, file := range duplicateFiles {
					fmt.Println(i+1, file)
				}

				fmt.Printf("Which file to keep? ")
				input, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Invalid input")
					continue
				}

				input = strings.TrimRight(input, "\n\r")
				intInput, err := strconv.Atoi(input)
				if err != nil || intInput > len(duplicateFiles) || intInput < 1 {
					fmt.Println("Invalid input")
					continue
				}

				for i, file := range duplicateFiles {
					if i+1 == intInput {
						continue
					}

					if *force {
						delete(file)
					}
				}

			}
		}
	} else {
		for size := range filesMap.FilesBySize {
			for hash := range filesMap.FilesBySize[size] {
				duplicateFiles := filesMap.FilesBySize[size][hash]
				if len(duplicateFiles) <= 1 {
					continue
				}

				for _, file := range duplicateFiles {
					fmt.Println(file)
				}
				fmt.Println()
			}
		}
	}
}

func printConfiguration() {
	fmt.Printf("fromFile: \"%v\"\n", *fromFile)
	fmt.Printf("toFile: \"%v\"\n", *toFile)
	fmt.Printf("deleteDupesIn: \"%v\"\n", *deleteDupesIn)
	fmt.Printf("force: \"%v\"\n", *force)
	fmt.Println("Searching paths:")
	for _, path := range flag.Args() {
		fmt.Println("- ", path)
	}

	fmt.Println()
	fmt.Println()
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
