package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wlmsrvty/git-go/mygit"
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
	{Name: "write-tree",
		Run: writeTree},
	{Name: "commit-tree",
		Run: commitTree},
	{Name: "clone",
		Run: clone},
	{Name: "ls-remote",
		Run: lsRemote},
	{Name: "log",
		Run: logCommit},
	{Name: "commit",
		Run: commit},
}

func Usage() {
	usage := `Usage: mygit <command> [<args>...]

Commands:
    init        Initialize the git directory structure
    cat-file    Provide content or type and size information for repository objects
    hash-object Compute object ID and optionally creates a blob from a file
    ls-tree 	List the contents of a tree object
    write-tree 	Create a tree object from the current working directory
    commit-tree Create a new commit object
    clone       Clone a repository into a new directory
    ls-remote   List references in a remote repository
    log         Show commit logs for a commit ID
    commit      Record changes to the repository`
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

func writeTree(args []string) error {
	flagSet := flag.NewFlagSet("write-tree", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Create a tree object from the current working directory

Usage: mygit write-tree`)
	}

	flagSet.Parse(args)

	treeEntry, err := mygit.RecordTree(".", true)
	if err != nil {
		return err
	}

	fmt.Println(treeEntry.Hash)

	return nil
}

func commitTree(args []string) error {
	flagSet := flag.NewFlagSet("commit-tree", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Create a new commit object

Usage: mygit commit-tree [options] <tree_sha>

Options:
    -p <parent_commit>  Parent commit hash
    -m <message>        Commit message`)
	}

	var parentCommit string
	flagSet.StringVar(&parentCommit, "p", "", "Parent commit")
	var commitMessage string
	flagSet.StringVar(&commitMessage, "m", "", "Commit message")

	// workaround for flag package not supporting options after arguments
	var treeSha string
	if len(args) > 1 {
		switch args[0] {
		case "-p":
			flagSet.Parse(args)
		case "-m":
			flagSet.Parse(args)
		default:
			treeSha = args[0]
			flagSet.Parse(args[1:])
		}
	}

	flagSet.Parse(args)

	if flagSet.NArg() < 1 && treeSha == "" {
		flagSet.Usage()
		os.Exit(1)
	}

	if treeSha == "" {
		treeSha = flagSet.Arg(0)
	}

	if commitMessage == "" {
		fmt.Fprintln(os.Stderr, "Commit message is required")
		flagSet.Usage()
		os.Exit(1)
	}

	commitOID, err := mygit.CommitTree(treeSha, parentCommit, commitMessage)
	if err != nil {
		return err
	}

	fmt.Println(commitOID)

	return nil
}

func clone(args []string) error {
	flagSet := flag.NewFlagSet("clone", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Clone a repository into a new directory

Usage: mygit clone <url> [<directory>]`)
	}
	flagSet.Parse(args)

	if flagSet.NArg() < 1 {
		flagSet.Usage()
		os.Exit(1)
	}

	url := flagSet.Arg(0)
	repoName := ""
	if flagSet.NArg() >= 2 {
		repoName = flagSet.Arg(1)
	}

	err := mygit.Clone(url, repoName)

	return err
}

func lsRemote(args []string) error {
	flagSet := flag.NewFlagSet("ls-remote", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`List references in a remote repository

Usage: mygit ls-remote <url>`)
	}
	flagSet.Parse(args)

	if flagSet.NArg() < 1 {
		flagSet.Usage()
		os.Exit(1)
	}

	url := flagSet.Arg(0)
	err := mygit.DisplayRemoteRefs(url)

	return err
}

func logCommit(args []string) error {
	flagSet := flag.NewFlagSet("log", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Show commit logs for a commit ID

Usage: mygit log [<commit_id>]`)
	}
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	commitId := ""
	if flagSet.NArg() > 0 {
		commitId = flagSet.Arg(0)
	}
	if err := mygit.Log(commitId); err != nil {
		return err
	}

	return nil
}

func commit(args []string) error {
	flagSet := flag.NewFlagSet("commit", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr,
			`Record changes to the repository

Usage: mygit commit -m <message>`)
	}

	var message string
	flagSet.StringVar(&message, "m", "", "Commit message")

	var allowEmptyMessage bool
	flagSet.BoolVar(&allowEmptyMessage, "allow-empty-message", false, "Allow empty message")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if message == "" && !allowEmptyMessage {
		return fmt.Errorf("commit message is required")
	}

	if err := mygit.Commit(message); err != nil {
		return err
	}

	return nil
}
