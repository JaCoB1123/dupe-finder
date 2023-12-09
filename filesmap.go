package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sync"

	"github.com/vbauerster/mpb/v8"
)

// FilesMap is a struct for listing files by Size and Hash to search for duplicates
type FilesMap struct {
	Images          []imageEntry
	FilesBySize     map[int64]string
	FilesByHash     map[string][]string
	FilesHashing    chan fileEntry
	FilesHashed     chan fileEntry
	ImagesHashing   chan imageEntry
	ImagesHashed    chan imageEntry
	progress        *mpb.Progress
	incomingBar     *mpb.Bar
	fileHashingBar  *mpb.Bar
	imageHashingBar *mpb.Bar
	lock            sync.Mutex
}

func newFilesMap() *FilesMap {
	return &FilesMap{
		FilesBySize:   map[int64]string{},
		FilesByHash:   map[string][]string{},
		FilesHashed:   make(chan fileEntry, 100000),
		FilesHashing:  make(chan fileEntry),
		ImagesHashed:  make(chan imageEntry, 100000),
		ImagesHashing: make(chan imageEntry),
		progress:      mpb.New(mpb.WithWidth(64)),
	}
}

func (fm *FilesMap) FileHashingWorker(wg *sync.WaitGroup) {
	for file := range fm.FilesHashing {
		if *verbose {
			fmt.Println("Hashing file", file.path)
		}

		hash, err := calculateFileHash(file.path)
		fm.fileHashingBar.IncrInt64(file.size)
		fm.FilesHashed <- file

		if err != nil {
			fmt.Fprintf(fm.progress, "Error calculating Hash for file %s: %v\n", file.path, err)
			continue
		}

		file.hash = hash
	}
	wg.Done()
}

func (fm *FilesMap) ImageHashingWorker(wg *sync.WaitGroup) {
	for file := range fm.ImagesHashing {
		if *verbose {
			fmt.Println("Hashing image", file.path)
		}

		hash, err := calculateImageHash(file.path)
		fm.imageHashingBar.IncrInt64(file.size)

		if errors.Is(err, image.ErrFormat) {
			continue
		} else if err != nil {
			fmt.Fprintf(fm.progress, "Error calculating Hash for image %s: %v\n", file.path, err)
			continue
		}

		file.imageHash = hash
		fm.ImagesHashed <- file
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

	for file := range fm.ImagesHashed {
		if *verbose {
			fmt.Println("Finishing", file.path)
		}

		fm.lock.Lock()
		fm.Images = append(fm.Images, file)
		fm.lock.Unlock()
	}

	done <- true
}

func (fm *FilesMap) WalkDirectories() int64 {
	var countFiles int64 = 0
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

			countFiles++
			fm.incomingBar.SetTotal(int64(countFiles), false)

			fm.hashImage(path, size)
			count := fm.hashFile(path, size)
			if count > 0 {
				sumSize += size * count
				fm.fileHashingBar.SetTotal(int64(sumSize), false)
			}
			return nil
		})
	}

	fm.incomingBar.SetTotal(int64(countFiles), true)
	close(fm.FilesHashing)
	close(fm.ImagesHashing)
	return countFiles
}

func (fm *FilesMap) hashFile(path string, size int64) int64 {
	prevFile, ok := fm.FilesBySize[size]
	if !ok {
		fm.FilesBySize[size] = path
		return 0
	}

	fm.FilesBySize[size] = ""
	fm.incomingBar.Increment()
	if *verbose {
		fmt.Println("Incoming", path)
	}

	fm.FilesHashing <- fileEntry{path, size, ""}
	if prevFile != "" {
		fm.FilesHashing <- fileEntry{prevFile, size, ""}
		return 2
	}

	return 1
}

func (fm *FilesMap) hashImage(path string, size int64) {
	fm.ImagesHashing <- imageEntry{path, size, 0}
}

type imageEntry struct {
	path      string
	size      int64
	imageHash uint64
}

type fileEntry struct {
	path string
	size int64
	hash string
}
