package main

import (
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/corona10/goimagehash"
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

func calculateHash(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	if strings.HasSuffix(path, ".jpg") {
		img, err := jpeg.Decode(f)
		if err != nil {
			return 0, err
		}
		hash, err := goimagehash.DifferenceHash(img)
		if err != nil {
			return 0, err
		}

		return hash.GetHash(), nil
	}

	/*	h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			return 0, err
		}

		return base64.RawStdEncoding.EncodeToString(h.Sum(nil)), nil*/
	return 0, nil
}
