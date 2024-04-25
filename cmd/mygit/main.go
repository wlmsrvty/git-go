package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
)

func helpUsage() {
	usage := []string{
		"usage: mygit <command> [<args>...]",
		"\tmygit init",
		"\tmygit cat-file -p <blob_sha>",
		"\tmygit hash-object -w <file_path>",
	}
	for _, line := range usage {
		fmt.Fprintf(os.Stderr, "%s\n", line)
	}
}

// Usage: your_git.sh <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 2 {
		helpUsage()
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		git_init()

	case "cat-file":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: mygit cat-file <blob_sha>\n")
			os.Exit(1)
		}
		git_cat_file(os.Args[2:])

	case "hash-object":
		git_hash_object(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func git_init() {
	/*
	   Initialize the git directory structure
	   https://git-scm.com/book/en/v2/Git-Internals-Git-Objects

	   .git/
	   .git/objects/
	   .git/refs/
	*/

	dirs := []string{".git", ".git/objects", ".git/refs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		}
	}

	headFileContents := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
	}

	fmt.Println("Initialized git directory")
}

func git_cat_file_error() {
	fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <blob_sha>\n")
	os.Exit(1)
}

func git_cat_file(Args []string) {
	if len(Args) < 2 || Args[0] != "-p" {
		git_cat_file_error()
	}
	blob_sha := Args[1]
	path := fmt.Sprintf(".git/objects/%s/%s", blob_sha[:2], blob_sha[2:])
	if _, err := os.Stat(path); err != nil {
		fmt.Fprintf(os.Stderr, "File not found: %s\n", path)
		os.Exit(1)
	}
	data, err := os.ReadFile(path)
	check(err)
	bytes := bytes.NewBuffer(data)
	zlibReader, err := zlib.NewReader(bytes)
	check(err)
	bufioReader := bufio.NewReader(zlibReader)
	// skip until the first null byte
	for {
		b, err := bufioReader.ReadByte()
		check(err)
		if b == 0 {
			break
		}
	}
	_, err = io.Copy(os.Stdout, bufioReader)
	check(err)
	zlibReader.Close()
}

func git_hash_object(Args []string) {
	// writeOption := Args[0] // -w
	filePath := Args[1]
	fileInfo, err := os.Stat(filePath)
	check(err)

	header := fmt.Sprintf("blob %d\000", fileInfo.Size())

	h := sha1.New()

	f, err := os.Open(filePath)
	check(err)
	defer f.Close()

	_, err = h.Write([]byte(header))
	check(err)

	_, err = io.Copy(h, f)
	check(err)

	shaString := fmt.Sprintf("%x", h.Sum(nil))
	objectPath := fmt.Sprintf(".git/objects/%s/%s", shaString[:2], shaString[2:])

	if _, err := os.Stat(objectPath); errors.Is(err, os.ErrNotExist) {
		objectFile, err := os.Create(objectPath)
		check(err)
		defer objectFile.Close()

		zlibWriter := zlib.NewWriter(f)
		defer zlibWriter.Close()

		zlibWriter.Write([]byte(header))

		f.Seek(0, 0)
		io.Copy(zlibWriter, f)
	}

	fmt.Printf("%x\n", h.Sum(nil))
}
