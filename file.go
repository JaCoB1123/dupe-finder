package main

import (
	"os"
	"path/filepath"
	"strconv"
)

func remove(path string) {
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
