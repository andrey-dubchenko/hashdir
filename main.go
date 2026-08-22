package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
)

const (
	version      = "1.0.0"
	author       = "Andrey Dubchenko"
	authorURL    = "https://github.com/andrey-dubchenko"
	defaultHash  = "sha256"
	usageMessage = "Usage: hashdir [sha256/md5] <directory>"
)

// hashers maps a supported algorithm name to its constructor.
var hashers = map[string]func() hash.Hash{
	"md5":    md5.New,
	"sha256": sha256.New,
}

func calculateHash(newHash func() hash.Hash, filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer closeQuietly(file)

	h := newHash()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// walkFiles hashes every regular file under root and writes "path: hash" lines
// to outputFile. skipPath is excluded so the output file never hashes itself.
func walkFiles(newHash func() hash.Hash, root string, skipPath string, outputFile *os.File) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() || path == skipPath {
			return nil
		}

		fileHash, err := calculateHash(newHash, path)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(outputFile, "%s: %s\n", path, fileHash)
		return err
	})
}

func closeQuietly(f *os.File) {
	if err := f.Close(); err != nil {
		fmt.Println("Error while closing file:", err)
	}
}

func run(algorithm, dir string) error {
	newHash, ok := hashers[algorithm]
	if !ok {
		return fmt.Errorf("bad algorithm %q, use sha256 or md5", algorithm)
	}

	outputPath, err := filepath.Abs(filepath.Join(dir, algorithm+"_hashes.txt"))
	if err != nil {
		return fmt.Errorf("error while resolving output path: %w", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error while creating file with hashes: %w", err)
	}
	defer closeQuietly(outputFile)

	fmt.Printf("Calculating hashes for directory -> %s...\n", dir)

	if err := walkFiles(newHash, dir, outputPath, outputFile); err != nil {
		return fmt.Errorf("error while trying to calculate hashes: %w", err)
	}

	fmt.Printf("Result saved to file %s\n", outputPath)
	return nil
}

func printVersion() {
	fmt.Printf("hashdir version %s\n", version)
	fmt.Printf("Author: %s - %s\n", author, authorURL)
}

// parseArgs resolves the algorithm and target directory from CLI arguments,
// defaulting to sha256 when only a directory is given.
func parseArgs(args []string) (algorithm string, dir string, ok bool) {
	switch len(args) {
	case 1:
		fmt.Printf("Algorithm not specified, using %s by default\n", defaultHash)
		return defaultHash, args[0], true
	case 2:
		return args[0], args[1], true
	default:
		return "", "", false
	}
}

func main() {
	args := os.Args[1:]

	if len(args) == 1 && (args[0] == "-v" || args[0] == "--version") {
		printVersion()
		return
	}

	algorithm, dir, ok := parseArgs(args)
	if !ok {
		fmt.Println(usageMessage)
		return
	}

	if err := run(algorithm, dir); err != nil {
		fmt.Println(err)
	}
}
