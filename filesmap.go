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
	Images      []*imageEntry
	FilesBySize map[int64]string
	FilesByHash map[string][]string
	// fileSizes records the size of every file found during the walk, so image
	// candidates can be hashed without re-stat'ing the files. It is not
	// serialized to the results file.
	fileSizes       map[string]int64
	FilesHashing    chan *fileEntry
	FilesHashed     chan *fileEntry
	ImagesHashing   chan *imageEntry
	ImagesHashed    chan *imageEntry
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
		fileSizes:     map[string]int64{},
		FilesHashing:  make(chan *fileEntry),
		FilesHashed:   make(chan *fileEntry, 100000),
		ImagesHashing: make(chan *imageEntry),
		ImagesHashed:  make(chan *imageEntry, 100000),
		progress:      mpb.New(mpb.WithWidth(64)),
	}
}

// Stage 2: compute the file hash of every file sent over FilesHashing.
func (fm *FilesMap) FileHashingWorker(wg *sync.WaitGroup) {
	defer wg.Done()
	for file := range fm.FilesHashing {
		if *verbose {
			fmt.Fprintf(fm.progress, "Hashing file %s\n", file.path)
		}

		hash, err := calculateFileHash(file.path)
		fm.fileHashingBar.IncrInt64(file.size)
		if err != nil {
			fmt.Fprintf(fm.progress, "Error calculating Hash for file %s: %v\n", file.path, err)
			continue
		}

		file.hash = hash
		fm.FilesHashed <- file
	}
}

// Stage 2: collect the file hashes into FilesByHash once FilesHashed is closed.
func (fm *FilesMap) fileHashMergeWorker(done chan bool) {
	for file := range fm.FilesHashed {
		if *verbose {
			fmt.Printf("Finishing '%s' with hash '%s'\n", file.path, file.hash)
		}

		fm.lock.Lock()
		fm.FilesByHash[file.hash] = append(fm.FilesByHash[file.hash], file.path)
		fm.lock.Unlock()
	}
	done <- true
}

// Stage 4: compute the image hash of every candidate sent over ImagesHashing.
func (fm *FilesMap) ImageHashingWorker(wg *sync.WaitGroup) {
	defer wg.Done()
	for file := range fm.ImagesHashing {
		if *verbose {
			fmt.Fprintf(fm.progress, "Hashing image %s\n", file.path)
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
}

// Stage 4: collect the image hashes into Images once ImagesHashed is closed.
func (fm *FilesMap) imageHashMergeWorker(done chan bool) {
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

// Stage 1: walk the trees and deduplicate by file size.
// A file is only file-hashed once another file with the same size has been
// seen; a unique size cannot be an exact duplicate. It returns the number of
// files found.
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

			fm.fileSizes[path] = size
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
	return countFiles
}

// hashFile applies the file size step for a single file. It returns how many
// files were queued to be file-hashed.
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

	fm.FilesHashing <- &fileEntry{path, size, ""}
	if prevFile != "" {
		fm.FilesHashing <- &fileEntry{prevFile, size, ""}
		return 2
	}

	return 1
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
