package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

func promptForDeletion(reader *bufio.Reader, files []string) {
	fmt.Print("\033[H\033[2J")
	for i, file := range files {
		fmt.Println(i+1, file)
	}
	fmt.Println(0, "Keep all")

	fmt.Printf("Which file to keep? ")
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Invalid input")
		return
	}

	input = strings.TrimRight(input, "\n\r")
	intInput, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Invalid input")
		return
	}

	if intInput == 0 {
		return
	}

	if intInput > len(files) || intInput < 1 {
		fmt.Println("Invalid input")
		return
	}

	for i, file := range files {
		if i+1 == intInput {
			continue
		}

		if *force {
			remove(file)
		}
	}
}
