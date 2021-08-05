package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

// FilesMap is a struct for listing files by Size and Hash to search for duplicates
type FilesMap struct {
	FilesBySize map[int64]string

	FilesByHash map[string][]string

	FilesHashing chan fileEntry

	FilesHashed chan fileEntry

	progress *mpb.Progress

	incomingBar *mpb.Bar

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
	fm.incomingBar = fm.progress.AddSpinner(0,
		mpb.PrependDecorators(
			decor.Name("Finding files "),
			decor.Elapsed(decor.ET_STYLE_HHMMSS),
		),
		mpb.AppendDecorators(
			decor.AverageSpeed(0, "%f   "),
			decor.CountersNoUnit("%d / %d"),
		),
	)
	for _, path := range flag.Args() {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}

			size := info.Size()
			if *minSize > size {
				return nil
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
				fm.FilesHashing <- fileEntry{prevFile, size, ""}
			}

			fm.FilesBySize[size] = ""

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
