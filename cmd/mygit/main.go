package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

type Command struct {
	Name string
	Run  func(args []string) error
}

var commands = []Command{
	{Name: "init",
		Run: git_init},
	{Name: "cat-file",
		Run: git_cat_file},
	{Name: "hash-object",
		Run: git_hash_object},
}

func Usage() {
	usage := `Usage: mygit <command> [<args>...]

Commands:
    init        Initialize the git directory structure
    cat-file    Provide content or type and size information for repository objects
    hash-object Compute object ID and optionally creates a blob from a file
    `
	fmt.Fprintf(os.Stderr, "%s", usage)
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	subCmd := flag.Arg(0)
	subCmdArgs := flag.Args()[1:]

	for _, cmd := range commands {
		if cmd.Name == subCmd {
			if err := cmd.Run(subCmdArgs); err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown command %s\n", subCmd)
	flag.Usage()
	os.Exit(1)
}

func git_init(args []string) error {
	flagSet := flag.NewFlagSet("init", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Create an empty Git repository or reinitialize an existing one

        Usage: mygit init`)
	}
	flagSet.Parse(args)

	dirs := []string{".git", ".git/objects", ".git/refs"}
	existing := false
	for _, dir := range dirs {
		if stat, err := os.Stat(dir); !os.IsNotExist(err) {
			if stat.IsDir() {
				existing = true
			} else {
				return fmt.Errorf("existing %s is not a directory", dir)
			}
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	headFileContents := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
		return err
	}

	path, err := os.Getwd()
	if err != nil {
		return err
	}

	if existing {
		fmt.Printf("Reinitialized existing Git repository in %s\n", path)
	} else {
		fmt.Printf("Initialized empty Git repository in %s/.git/\n", path)
	}

	return nil
}

func git_cat_file(args []string) error {
	var prettyPrint bool
	flagSet := flag.NewFlagSet("cat-file", flag.ExitOnError)
	flagSet.BoolVar(&prettyPrint, "p", false,
		"Pretty-print the contents of the object to the terminal")
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Provide content or type and size information for repository objects

Usage: mygit cat-file [options] <blob_sha>`)
		flagSet.PrintDefaults()
	}
	flagSet.Parse(args)

	if flagSet.NArg() < 1 {
		flagSet.Usage()
		os.Exit(1)
	}

	blob_sha := flagSet.Arg(0)

	// Check if the object exists
	path := fmt.Sprintf(".git/objects/%s/%s", blob_sha[:2], blob_sha[2:])
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("object file not found: %s", path)
	}

	// Read the object
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Get bytes buffer from data
	bytes := bytes.NewBuffer(data)

	// Decompress the object using zlib reader
	zlibReader, err := zlib.NewReader(bytes)
	if err != nil {
		return err
	}
	defer zlibReader.Close()

	// Use a new reader to skip the header
	bufioReader := bufio.NewReader(zlibReader)
	// skip until the first null byte
	for {
		b, err := bufioReader.ReadByte()
		if err != nil {
			return err
		}
		if b == 0 {
			break
		}
	}

	// Pretty print the contents of the object to the terminal
	_, err = io.Copy(os.Stdout, bufioReader)
	if err != nil {
		return err
	}

	return nil
}

func git_hash_object(args []string) error {
	var writeOption bool
	flagSet := flag.NewFlagSet("hash-object", flag.ExitOnError)
	flagSet.BoolVar(&writeOption, "w", false,
		"Actually write the object into the database")

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr,
			`Compute object ID and optionally creates a blob from a file

Usage: mygit hash-object [options] <file>`)
		flagSet.PrintDefaults()
	}

	flagSet.Parse(args)

	if flagSet.NArg() < 1 {
		flagSet.Usage()
		os.Exit(1)
	}

	filePath := flagSet.Arg(0)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("blob %d\000", fileInfo.Size())

	hash := sha1.New()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = hash.Write([]byte(header)); err != nil {
		return err
	}

	if _, err = io.Copy(hash, file); err != nil {
		return err
	}

	shaString := fmt.Sprintf("%x", hash.Sum(nil))

	fmt.Printf("%s\n", shaString)

	if !writeOption {
		return nil
	}

	objectFolderPath := fmt.Sprintf(".git/objects/%s", shaString[:2])
	objectPath := fmt.Sprintf(".git/objects/%s/%s", shaString[:2], shaString[2:])

	if _, err := os.Stat(objectPath); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(objectFolderPath, 0755); !errors.Is(err, os.ErrExist) {
			return err
		}

		objectFile, err := os.Create(objectPath)
		if err != nil {
			return err
		}
		defer objectFile.Close()

		zlibWriter := zlib.NewWriter(objectFile)
		defer zlibWriter.Close()

		zlibWriter.Write([]byte(header))

		file.Seek(0, 0)
		io.Copy(zlibWriter, file)
	}

	return nil
}
