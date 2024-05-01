package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/codecrafters-io/git-starter-go/mygit"
)

type Command struct {
	Name string
	Run  func(args []string) error
}

var commands = []Command{
	{Name: "init",
		Run: gitInit},
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
    hash-object Compute object ID and optionally creates a blob from a file
	ls-tree 	List the contents of a tree object`
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
				fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown command %s\n", subCmd)
	flag.Usage()
	os.Exit(1)
}

func gitInit(args []string) error {
	flagSet := flag.NewFlagSet("init", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Create an empty Git repository or reinitialize an existing one

        Usage: mygit init`)
	}
	flagSet.Parse(args)

	return mygit.Initialize()
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

	gitObject, err := mygit.NewObject(blob_sha)
	if err != nil {
		return err
	}

	if prettyPrint {
		gitObject.CatFile()
		return nil
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

	options := mygit.HashObjectOptions{
		Path:  filePath,
		Write: writeOption,
	}
	return mygit.HashObject(&options)
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
	gitObject, err := mygit.NewObject(tree_sha)
	if err != nil {
		return err
	}

	options := mygit.PrintTreeContentOptions{
		NameOnly: nameOnlyOption,
	}

	return gitObject.PrintTreeContent(&options)
}
