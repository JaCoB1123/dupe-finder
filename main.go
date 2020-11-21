package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

func main() {
	for _, path := range os.Args[1:] {
		fmt.Println(path)
	}

}

type filesMap struct {
	Files map[int]map[[32]byte]*fileEntry
}

type fileEntry struct {
	Path string
	Size int
}

func calculateHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(h.Sum(nil)), nil
}
