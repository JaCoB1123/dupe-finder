package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vbauerster/mpb/v7"
)

// FilesMap is a struct for listing files by Size and Hash to search for duplicates
type FilesMap struct {
	FilesBySize map[int64]string

	FilesByHash map[string][]string

	FilesHashing chan fileEntry

	FilesHashed chan fileEntry

	progress *mpb.Progress

	incomingBar *mpb.Bar

	hashingBar *mpb.Bar

	lock sync.Mutex
}

func newFilesMap() *FilesMap {
	return &FilesMap{
		FilesBySize:  map[int64]string{},
		FilesByHash:  map[string][]string{},
		FilesHashed:  make(chan fileEntry, 100000),
		FilesHashing: make(chan fileEntry),
		progress:     mpb.New(mpb.WithWidth(64)),
	}
}

func (fm *FilesMap) HashingWorker(wg *sync.WaitGroup) {
	for file := range fm.FilesHashing {
		if *verbose {
			fmt.Println("Hashing", file.path)
		}

		hash, err := calculateHash(file.path)

		if err != nil {
			log.Printf("Error calculating Hash for %s: %v\n", file.path, err)
			continue
		}

		file.hash = hash
		fm.hashingBar.IncrInt64(file.size)
		fm.FilesHashed <- file
	}
	wg.Done()
}

func (fm *FilesMap) HashedWorker(done chan bool) {
	for file := range fm.FilesHashed {
		if *verbose {
			fmt.Println("Finishing", file.path)
		}

		fm.lock.Lock()
		fm.FilesByHash[file.hash] = append(fm.FilesByHash[file.hash], file.path)
		fm.lock.Unlock()
	}

	done <- true
}

func (fm *FilesMap) WalkDirectories() int {
	countFiles := 0
	sumSize := int64(0)
	for _, path := range flag.Args() {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			size := info.Size()
			if *minSize > size {
				return nil
			}

			if !strings.HasSuffix(path, ".jpg") {
				size = 123456789123456
			}

			fm.incomingBar.Increment()
			countFiles++
			fm.incomingBar.SetTotal(int64(countFiles), false)
			if *verbose {
				fmt.Println("Incoming", path)
			}

			prevFile, ok := fm.FilesBySize[size]
			if !ok {
				fm.FilesBySize[size] = path
				return nil
			}

			if prevFile != "" {
				sumSize += size
				fm.FilesHashing <- fileEntry{prevFile, size, ""}
			}

			fm.FilesBySize[size] = ""

			sumSize += size
			fm.hashingBar.SetTotal(int64(sumSize), false)
			fm.FilesHashing <- fileEntry{path, info.Size(), ""}
			return nil
		})
	}

	fm.incomingBar.SetTotal(int64(countFiles), true)
	close(fm.FilesHashing)
	return countFiles
}

type fileEntry struct {
	path string
	size int64
	hash string
}
