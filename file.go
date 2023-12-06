package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"

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

func calculateImageHash(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	fmt.Println("jh:", path, err)
	img, _, err := image.Decode(f)
	fmt.Println("oh:", path, err)
	if err != nil {
		fmt.Println("kh:", path, err)
		return 0, err
	}
	fmt.Println("lh:", path, err)
	hash, err := goimagehash.DifferenceHash(img)
	if err != nil {
		fmt.Println("mh:", path, err)
		return 0, err
	}

	fmt.Println("nh:", path, err)
	fmt.Println(path, hash.ToString())
	return hash.GetHash(), nil
}

func calculateFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	stringHash := base64.RawStdEncoding.EncodeToString(h.Sum(nil))
	fmt.Println(path, stringHash)
	return stringHash, nil
}
