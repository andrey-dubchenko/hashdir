package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func readReport(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	res := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		i := strings.LastIndex(line, ": ")
		if i < 0 {
			t.Fatalf("malformed line %q", line)
		}
		res[line[:i]] = line[i+2:]
	}
	return res
}

func TestCalculateHash(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	writeFile(t, p, "hello")

	got, err := calculateHash(sha256.New, p)
	if err != nil || got != sha("hello") {
		t.Fatalf("sha256: got %q, %v", got, err)
	}

	sum := md5.Sum([]byte("hello"))
	got, err = calculateHash(md5.New, p)
	if err != nil || got != hex.EncodeToString(sum[:]) {
		t.Fatalf("md5: got %q, %v", got, err)
	}

	if _, err := calculateHash(sha256.New, p+".missing"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunSha256(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "bbb")

	if err := run("sha256", dir); err != nil {
		t.Fatal(err)
	}

	got := readReport(t, filepath.Join(dir, "sha256_hashes.txt"))
	want := map[string]string{
		filepath.Join(dir, "a.txt"):        sha("aaa"),
		filepath.Join(dir, "sub", "b.txt"): sha("bbb"),
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
}

func TestRunMd5(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")

	if err := run("md5", dir); err != nil {
		t.Fatal(err)
	}

	sum := md5.Sum([]byte("aaa"))
	got := readReport(t, filepath.Join(dir, "md5_hashes.txt"))
	if got[filepath.Join(dir, "a.txt")] != hex.EncodeToString(sum[:]) {
		t.Fatalf("unexpected report: %v", got)
	}
}

func TestRunBadAlgorithm(t *testing.T) {
	dir := t.TempDir()
	if err := run("crc32", dir); err == nil {
		t.Fatal("expected error")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("no files should be created, got %v", entries)
	}
}

func TestRunBadDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := run("sha256", filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected error for missing directory")
	}

	f := filepath.Join(dir, "file")
	writeFile(t, f, "x")
	if err := run("sha256", f); err == nil {
		t.Fatal("expected error for non-directory")
	}
}

func TestRunExcludesReportsAndIsRepeatable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")

	if err := run("md5", dir); err != nil {
		t.Fatal(err)
	}
	if err := run("sha256", dir); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "sha256_hashes.txt"))

	if err := run("sha256", dir); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "sha256_hashes.txt"))

	if string(first) != string(second) {
		t.Fatalf("report not repeatable:\n%s\nvs\n%s", first, second)
	}
	if strings.Contains(string(second), "_hashes.txt") {
		t.Fatalf("report lists reports:\n%s", second)
	}
}

func TestRunRelativeDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")

	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := run("sha256", "."); err != nil {
		t.Fatal(err)
	}
	got := readReport(t, filepath.Join(dir, "sha256_hashes.txt"))
	if len(got) != 1 || got["a.txt"] != sha("aaa") {
		t.Fatalf("unexpected report: %v", got)
	}
}

func TestSymlinks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
	if err := os.Mkdir(filepath.Join(dir, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.txt", filepath.Join(dir, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink("d", filepath.Join(dir, "linkdir")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := run("sha256", dir); err != nil {
		t.Fatal(err)
	}
	got := readReport(t, filepath.Join(dir, "sha256_hashes.txt"))
	if got[filepath.Join(dir, "link.txt")] != sha("aaa") {
		t.Errorf("file symlink not hashed by target content: %v", got)
	}
	if _, ok := got[filepath.Join(dir, "linkdir")]; ok {
		t.Errorf("directory symlink must be skipped: %v", got)
	}
	if len(got) != 2 {
		t.Errorf("unexpected entries: %v", got)
	}
}

func TestBrokenSymlinkFailsAndRemovesReport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
	if err := os.Symlink("gone", filepath.Join(dir, "bad")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := run("sha256", dir); err == nil {
		t.Fatal("expected error for broken symlink")
	}
	if _, err := os.Stat(filepath.Join(dir, "sha256_hashes.txt")); !os.IsNotExist(err) {
		t.Fatalf("partial report must be removed, stat err: %v", err)
	}
}

func TestParseArgs(t *testing.T) {
	alg, dir, ok := parseArgs([]string{"x"})
	if !ok || alg != defaultHash || dir != "x" {
		t.Errorf("one arg: %q %q %v", alg, dir, ok)
	}
	alg, dir, ok = parseArgs([]string{"md5", "x"})
	if !ok || alg != "md5" || dir != "x" {
		t.Errorf("two args: %q %q %v", alg, dir, ok)
	}
	for _, args := range [][]string{nil, {"a", "b", "c"}} {
		if _, _, ok := parseArgs(args); ok {
			t.Errorf("args %v should be rejected", args)
		}
	}
}
