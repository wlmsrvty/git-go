package main

import (
	"bufio"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
	{Name: "ls-tree",
		Run: git_ls_tree},
}

func Usage() {
	usage := `Usage: mygit <command> [<args>...]

Commands:
    init        Initialize the git directory structure
    cat-file    Provide content or type and size information for repository objects
    hash-object Compute object ID and optionally creates a blob from a file`
	fmt.Fprintf(os.Stderr, "%s\n", usage)
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

func getObjectPath(sha string) string {
	return fmt.Sprintf(".git/objects/%s/%s", sha[:2], sha[2:])
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

	path := getObjectPath(blob_sha)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("object file not found: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decompress the object using zlib reader
	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		return err
	}
	defer zlibReader.Close()

	bufioReader := bufio.NewReader(zlibReader)

	_, _, err = parseHeader(bufioReader)
	if err != nil {
		return fmt.Errorf("invalid object format")
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
		fmt.Fprintln(os.Stderr,
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

type ObjectType string

const (
	ObjectTypeBlob ObjectType = "blob"
	ObjectTypeTree ObjectType = "tree"
)

type TreeEntry struct {
	Mode string
	Type ObjectType
	Hash string
	Name string
}

func parseHeader(objectReader *bufio.Reader) (ObjectType, int, error) {
	/*
		Header:
			<type> <size>\x00<content>
	*/

	header, err := objectReader.ReadString(0)
	if err != nil {
		return "", 0, err
	}

	split := strings.Split(header[:len(header)-1], " ")
	if len(split) != 2 {
		return "", 0, fmt.Errorf("invalid object format")
	}

	objType := ObjectType(split[0])
	objSize, err := strconv.Atoi(split[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid object size")
	}

	return objType, objSize, nil
}

func parseTree(objContent *bufio.Reader) ([]TreeEntry, error) {

	/*
		Format of the tree object:
			tree <size>\x00<entry><entry>...
		Format of the entry:
			<mode> <name>\x00<20_byte_hash>
	*/

	entries := []TreeEntry{}

	for {
		header, err := objContent.ReadString(0)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		split := strings.Split(header[:len(header)-1], " ")
		if len(split) != 2 {
			return nil, fmt.Errorf("invalid tree entry format")
		}

		mode := split[0]
		name := split[1]

		hash := make([]byte, 20)
		if _, err := objContent.Read(hash); err != nil {
			return nil, err
		}

		objType := ObjectTypeBlob
		if mode == "40000" {
			objType = ObjectTypeTree
		}

		entries = append(entries, TreeEntry{
			Mode: mode,
			Type: objType,
			Hash: fmt.Sprintf("%x", hash),
			Name: name,
		})
	}

	return entries, nil
}

func git_ls_tree(args []string) error {
	flagSet := flag.NewFlagSet("ls-tree", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`List the contents of a tree object

Usage: mygit ls-tree [options] <tree_sha>`)
		flagSet.PrintDefaults()
	}

	var nameOnlyOption bool
	flagSet.BoolVar(&nameOnlyOption, "name-only", false, "List only filenames")

	flagSet.Parse(args)

	if flagSet.NArg() < 1 {
		flagSet.Usage()
		os.Exit(1)
	}

	tree_sha := flagSet.Arg(0)
	path := getObjectPath(tree_sha)

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		return err
	}
	defer zlibReader.Close()

	bufioReader := bufio.NewReader(zlibReader)

	objType, _, err := parseHeader(bufioReader)
	if err != nil {
		return err
	}

	if objType != ObjectTypeTree {
		return fmt.Errorf("%s is not a tree object", tree_sha)
	}

	treeEntries, err := parseTree(bufioReader)
	if err != nil {
		return err
	}

	for _, treeEntry := range treeEntries {
		fmt.Println(treeEntry.Name)
	}

	return nil
}
