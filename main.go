package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func calculateHash(algorithm string, filename string) (string, error) {
	file, err := os.Open(filename)

	if err != nil {
		return "", err
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error while closing file:", err)
		}
	}(file)

	if algorithm == "md5" {
		hash := md5.New()

		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}

		return hex.EncodeToString(hash.Sum(nil)), nil
	}

	if algorithm == "sha256" {
		hash := sha256.New()

		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}

		return hex.EncodeToString(hash.Sum(nil)), nil
	}

	return "", nil
}

func walkFiles(algorithm string, root string, outputFile *os.File) error {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		hash, err := calculateHash(algorithm, path)

		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(outputFile, "%s: %s\n", path, hash)

		return err
	})

	return err
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: sha256/md5 <directory>")

		return
	}

	algorithm := os.Args[1]
	dir := os.Args[2]
	outputFilename := ""

	if algorithm != "sha256" && algorithm != "md5" {
		fmt.Println("Bad algorithm. Use sha256 or md5.")

		return
	}

	if algorithm == "sha256" {
		outputFilename = "sha256_hashes.txt"
	}

	if algorithm == "md5" {
		outputFilename = "md5_hashes.txt"
	}

	filePath := dir + "\\" + outputFilename

	outputFile, err := os.Create(filePath)

	if err != nil {
		fmt.Println("Error while creating file with hashes:", err)

		return
	}

	defer func(outputFile *os.File) {
		err := outputFile.Close()
		if err != nil {
			fmt.Println("Error while closing file:", err)
		}
	}(outputFile)

	fmt.Printf("Calculating hashes for directory ->  %s...\n", dir)

	if err := walkFiles(algorithm, dir, outputFile); err != nil {
		fmt.Println("Error while trying to calculate hashes:", err)

		return
	}

	fmt.Printf("Result saved to file %s\n", filePath)
}
