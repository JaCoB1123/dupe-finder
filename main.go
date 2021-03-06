package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
)

var fromFile = flag.String("from-file", "", "Load results file from <path>")
var toFile = flag.String("to-file", "", "Save results to <path>")
var deleteDupesIn = flag.String("delete-dupes-in", "", "Delete duplicates if they are contained in <path>")
var promptForDelete = flag.Bool("delete-prompt", false, "Ask which file to keep for each dupe-set")
var moveToFolder = flag.String("move-files", "", "Move files to <path> instead of deleting them")
var force = flag.Bool("force", false, "Actually delete files. Without this options, the files to be deleted are only printed")
var verbose = flag.Bool("verbose", false, "Output additional information")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *verbose {
		printConfiguration()
	}
	countFiles := 0
	filesMap := newFilesMap()
	if *fromFile != "" {
		byteValue, _ := ioutil.ReadFile(*fromFile)
		err := json.Unmarshal(byteValue, &filesMap.FilesByHash)
		if err != nil {
			panic(err)
		}
	} else {
		done := make(chan bool)
		wg := sync.WaitGroup{}
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			wg.Add(1)
			go filesMap.HashingWorker(&wg)
		}

		go filesMap.IncomingWorker()

		go filesMap.HashedWorker(done)

		countFiles = filesMap.WalkDirectories()

		wg.Wait()
		close(filesMap.FilesHashed)
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

		countInstances := 0
		countDupeSets := 0
		for hash := range filesMap.FilesByHash {
			duplicateFiles := filesMap.FilesByHash[hash]
			if len(duplicateFiles) <= 1 {
				continue
			}

			countDupeSets++
			for _, file := range duplicateFiles {
				countInstances++
				fmt.Println(file)
			}
			fmt.Println()
		}

		fmt.Println("Statistics:")
		fmt.Println(countFiles, "Files")
		fmt.Println(len(filesMap.FilesBySize), "Unique Sizes")
		fmt.Println(len(filesMap.FilesByHash), "Unique Hashes")
		fmt.Println(countInstances, "Duplicate Files")
		fmt.Println(countDupeSets, "Duplicate Sets")
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
