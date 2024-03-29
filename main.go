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
	"strings"
	"sync"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var fromFile = flag.String("from-file", "", "Load results file from <path>")
var toFile = flag.String("to-file", "", "Save results to <path>")
var deleteDupesIn = flag.String("delete-dupes-in", "", "Delete duplicates if they are contained in <path>")
var promptForDelete = flag.Bool("delete-prompt", false, "Ask which file to keep for each dupe-set")
var moveToFolder = flag.String("move-files", "", "Move files to <path> instead of deleting them")
var minSize = flag.Int64("min-size", -1, "Ignore all files smaller than <size> in Bytes")
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

	var countFiles int64 = 0
	filesMap := newFilesMap()
	if *fromFile != "" {
		byteValue, _ := ioutil.ReadFile(*fromFile)
		err := json.Unmarshal(byteValue, &filesMap.FilesByHash)
		if err != nil {
			panic(err)
		}
	} else {
		filesMap.incomingBar = filesMap.progress.AddSpinner(0,
			mpb.PrependDecorators(
				decor.Name("Finding files  "),
				decor.Elapsed(decor.ET_STYLE_HHMMSS),
			),
			mpb.AppendDecorators(
				decor.AverageSpeed(0, "%8.2f"),
				decor.Name("   "),
				decor.CurrentNoUnit("%5d"),
			),
		)
		filesMap.fileHashingBar = filesMap.progress.AddBar(0,
			mpb.PrependDecorators(
				decor.Name("Hashing files  "),
				decor.Elapsed(decor.ET_STYLE_HHMMSS),
			),
			mpb.AppendDecorators(
				decor.AverageSpeed(decor.SizeB1024(0), "%23.2f"),
				decor.Name("   "),
				decor.CurrentKibiByte("%5d"),
			),
		)
		filesMap.imageHashingBar = filesMap.progress.AddBar(0,
			mpb.PrependDecorators(
				decor.Name("Hashing images "),
				decor.Elapsed(decor.ET_STYLE_HHMMSS),
			),
			mpb.AppendDecorators(
				decor.AverageSpeed(decor.SizeB1024(0), "%23.2f"),
				decor.Name("   "),
				decor.CurrentKibiByte("%5d"),
			),
		)
		done := make(chan bool)
		wg := sync.WaitGroup{}
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			wg.Add(2)
			go filesMap.ImageHashingWorker(&wg)
			go filesMap.FileHashingWorker(&wg)
		}

		go filesMap.HashedWorker(done)

		countFiles = filesMap.WalkDirectories()

		wg.Wait()
		close(filesMap.FilesHashed)
		close(filesMap.ImagesHashed)
		<-done
	}

	if *toFile != "" && *fromFile == "" {
		json, _ := json.MarshalIndent(filesMap.FilesByHash, "", "  ")
		ioutil.WriteFile(*toFile, json, 0644)
	}

	for hash, duplicateFiles := range filesMap.FilesByHash {
		if len(duplicateFiles) > 1 {
			continue
		}

		delete(filesMap.FilesByHash, hash)
	}

	if *deleteDupesIn != "" {
		deleteIn := filepath.Clean(*deleteDupesIn)
		for hash := range filesMap.FilesByHash {
			duplicateFiles := filesMap.FilesByHash[hash]
			hasDupesInFolder := false
			hasDupesOutsideFolder := false
			for _, file := range duplicateFiles {
				fileIsInFolder := strings.HasPrefix(filepath.Clean(file), deleteIn)
				hasDupesOutsideFolder = hasDupesOutsideFolder || !fileIsInFolder
				hasDupesInFolder = hasDupesInFolder || fileIsInFolder
			}

			if !hasDupesInFolder || !hasDupesOutsideFolder {
				if !hasDupesOutsideFolder {
					fmt.Println("Not deleting one of the following files, since all would be deleted")
				}
				if !hasDupesInFolder {
					fmt.Println("Not deleting one of the following files, since none are in the selected directory")
				}

				for _, file := range duplicateFiles {
					fmt.Println("-", file)
				}
				fmt.Println()
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
			promptForDeletion(reader, duplicateFiles)
		}
	} else {
		countInstances := 0
		countDupeSets := 0

		fmt.Println("Files that are binary identical:")
		for _, duplicateFiles := range filesMap.FilesByHash {
			countDupeSets++
			for _, file := range duplicateFiles {
				countInstances++
				fmt.Println(file)
			}
			fmt.Println()
		}

		fmt.Println("Images that are similar:")
		imageClusters := filesMap.getImageClusters()
		for _, cluster := range imageClusters {
			countDupeSets++
			for _, image := range cluster.images {
				countInstances++
				fmt.Println(image.path, image.distance)
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
