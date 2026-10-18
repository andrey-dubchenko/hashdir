package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
)

// version is set at build time: -ldflags "-X main.version=...".
var version = "dev"

const (
	author       = "Andrey Dubchenko"
	authorURL    = "https://github.com/andrey-dubchenko"
	defaultHash  = "sha256"
	usageMessage = "Usage: hashdir [sha256/md5] <directory>\nSymbolic links to files are hashed by target content; links to directories and other non-regular files are skipped."
)

// hashers maps a supported algorithm name to its constructor.
var hashers = map[string]func() hash.Hash{
	"md5":    md5.New,
	"sha256": sha256.New,
}

func reportName(algorithm string) string {
	return algorithm + "_hashes.txt"
}

func calculateHash(newHash func() hash.Hash, filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := newHash()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// isReport reports whether path is a hashes report produced by this tool
// (any algorithm) located directly in root, or the file currently being written.
func isReport(root, path string, info os.FileInfo, output os.FileInfo) bool {
	if os.SameFile(info, output) {
		return true
	}
	if filepath.Dir(path) != filepath.Clean(root) {
		return false
	}
	for algorithm := range hashers {
		if filepath.Base(path) == reportName(algorithm) {
			return true
		}
	}
	return false
}

// walkFiles hashes every regular file under root and writes "path: hash" lines
// to out. Reports of this tool are excluded so they never hash themselves or
// each other. Unreadable files are reported to stderr and counted, not fatal.
func walkFiles(newHash func() hash.Hash, root string, output os.FileInfo, out io.Writer) (failed int, err error) {
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Skipping:", err)
			failed++
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// Hash the content of the link target; links to directories are skipped.
			info, err = os.Stat(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Skipping:", err)
				failed++
				return nil
			}
		}

		if !info.Mode().IsRegular() || isReport(root, path, info, output) {
			return nil
		}

		fileHash, err := calculateHash(newHash, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Skipping:", err)
			failed++
			return nil
		}

		_, err = fmt.Fprintf(out, "%s: %s\n", path, fileHash)
		return err
	})
	return failed, err
}

func run(algorithm, dir string) (err error) {
	newHash, ok := hashers[algorithm]
	if !ok {
		return fmt.Errorf("bad algorithm %q, use sha256 or md5", algorithm)
	}

	st, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("cannot access directory: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	outputPath, err := filepath.Abs(filepath.Join(dir, reportName(algorithm)))
	if err != nil {
		return fmt.Errorf("error while resolving output path: %w", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error while creating file with hashes: %w", err)
	}
	// On any failure remove the partial report; close errors are not lost.
	defer func() {
		if cerr := outputFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("error while closing file with hashes: %w", cerr)
		}
		if err != nil {
			os.Remove(outputPath)
		}
	}()

	outputInfo, err := outputFile.Stat()
	if err != nil {
		return fmt.Errorf("error while reading output file info: %w", err)
	}

	fmt.Printf("Calculating hashes for directory -> %s...\n", dir)

	w := bufio.NewWriter(outputFile)
	failed, err := walkFiles(newHash, dir, outputInfo, w)
	if err != nil {
		return fmt.Errorf("error while trying to calculate hashes: %w", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("error while writing hashes: %w", err)
	}
	if failed > 0 {
		return fmt.Errorf("%d file(s) could not be hashed", failed)
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

	if len(args) == 1 {
		switch args[0] {
		case "-v", "--version":
			printVersion()
			return
		case "-h", "--help":
			fmt.Println(usageMessage)
			return
		}
	}

	algorithm, dir, ok := parseArgs(args)
	if !ok {
		fmt.Fprintln(os.Stderr, usageMessage)
		os.Exit(2)
	}

	if err := run(algorithm, dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
