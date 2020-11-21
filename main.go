package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
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

func main() {
	flag.Parse()

	if *verbose {
		printConfiguration()
	}

	filesMap := newFilesMap()
	if *fromFile != "" {
		byteValue, _ := ioutil.ReadFile(*fromFile)
		err := json.Unmarshal(byteValue, &filesMap.FilesByHash)
		if err != nil {
			panic(err)
		}
	} else {
		done := make(chan bool)
		//for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		go filesMap.HashingWorker()
		//}

		go filesMap.IncomingWorker()

		go filesMap.HashedWorker(done)

		for _, path := range flag.Args() {
			filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				if info.IsDir() {
					return nil
				}

				filesMap.FilesIncoming <- fileEntry{path, info.Size(), ""}
				return nil
			})
		}

		close(filesMap.FilesIncoming)
		<-done
	}

	if *toFile != "" && *fromFile == "" {
		json, _ := json.MarshalIndent(filesMap.FilesByHash, "", "  ")
		ioutil.WriteFile(*toFile, json, 644)
	}

	if *deleteDupesIn != "" {
		deleteIn := filepath.Clean(*deleteDupesIn)
		for hash := range filesMap.FilesByHash {
			duplicateFiles := filesMap.FilesByHash[hash]
			if len(duplicateFiles) <= 1 {
				continue
			}

			for _, file := range duplicateFiles {
				if strings.HasPrefix(filepath.Clean(file), deleteIn) {
					fmt.Println("Would delete ", file)
					if *force {
						remove(file)
					}
				}
			}
		}
	} else if *promptForDelete {
		reader := bufio.NewReader(os.Stdin)
		for hash := range filesMap.FilesByHash {
			duplicateFiles := filesMap.FilesByHash[hash]
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
					remove(file)
				}

			}
		}
	} else {
		for hash := range filesMap.FilesByHash {
			duplicateFiles := filesMap.FilesByHash[hash]
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

type fileEntry struct {
	path string
	size int64
	hash string
}
