package main

import (
	"crypto/sha1"
	"encoding/base64"
	"io"
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

func calculateHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(h.Sum(nil)), nil
}
