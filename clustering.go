package main

import (
	"slices"

	"github.com/steakknife/hamming"
)

type imageCluster struct {
	images []similarImage
}

type similarImage struct {
	path     string
	distance int
}

func (fm *FilesMap) getImageClusters() []imageCluster {
	var clusters []imageCluster

	for len(fm.Images) > 0 {
		file := fm.Images[0]
		fm.Images = slices.Delete(fm.Images, 0, 1)

		var currentCluster []similarImage
		currentCluster = append(currentCluster, similarImage{path: file.path})
		for otherIndex := len(fm.Images) - 1; otherIndex >= 0; otherIndex-- {
			otherFile := fm.Images[otherIndex]
			var distance = hamming.Uint64(file.imageHash, otherFile.imageHash)
			if distance > 5 {
				continue
			}

			fm.Images = slices.Delete(fm.Images, otherIndex, otherIndex+1)
			currentCluster = append(currentCluster, similarImage{path: otherFile.path, distance: distance})
		}

		if len(currentCluster) <= 1 {
			continue
		}

		clusters = append(clusters, imageCluster{images: currentCluster})
	}

	return clusters
}
