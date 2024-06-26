package mygit

import (
	"fmt"
	"os"
)

const master = "master"

type printOption int

const (
	noPrint  printOption = iota
	yesPrint printOption = iota
)

// Initialize creates the necessary directories and files for a new git repository
func Initialize() error {
	err := createInitStructure(yesPrint)
	if err != nil {
		return err
	}

	return nil
}

func createInitStructure(pOption printOption) error {
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

	headFileContents := []byte("ref: refs/heads/" + master + "\n")
	if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
		return err
	}

	path, err := os.Getwd()
	if err != nil {
		return err
	}

	if pOption == yesPrint {
		if existing {
			fmt.Printf("Reinitialized existing Git repository in %s\n", path)
		} else {
			fmt.Printf("Initialized empty Git repository in %s/.git/\n", path)
		}
	}

	return nil
}
