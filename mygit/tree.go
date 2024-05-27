package mygit

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// ObjectType represents the type of a git object
type ObjectType string

const (
	ObjectTypeBlob   ObjectType = "blob"
	ObjectTypeTree   ObjectType = "tree"
	ObjectTypeCommit ObjectType = "commit"
)

// TreeEntry represents an entry in a git tree object
type TreeEntry struct {
	Mode      string
	Type      ObjectType
	Hash      string
	HashBytes []byte
	Name      string
}

type PrintTreeContentOptions struct {
	NameOnly bool
}

// PrintTreeContent prints every entry in a tree object
func (o *Object) PrintTreeContent(options *PrintTreeContentOptions) error {
	if o.Type != ObjectTypeTree {
		return fmt.Errorf("object %s is not a tree", o.Hash)
	}

	reader := bufio.NewReader(bytes.NewBuffer(o.Content))
	bufioReader := bufio.NewReader(reader)
	treeEntries, err := parseTree(bufioReader)
	if err != nil {
		return err
	}

	for _, treeEntry := range treeEntries {
		if options.NameOnly {
			fmt.Println(treeEntry.Name)
		} else {
			fmt.Printf("%s %s %s\t%s\n", treeEntry.Mode, treeEntry.Type, treeEntry.Hash, treeEntry.Name)
		}
	}

	return nil
}

// parseTree parses all entries in a tree object
// Format of the tree object content:
//
//	<mode> <name>\x00<20_byte_hash>
func parseTree(objContent *bufio.Reader) ([]TreeEntry, error) {

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
			Mode:      mode,
			Type:      objType,
			Hash:      fmt.Sprintf("%x", hash),
			HashBytes: hash,
			Name:      name,
		})
	}

	return entries, nil
}

func writeTreeToDisk(treeObject *Object, treePath string) error {
	if treeObject.Type != ObjectTypeTree {
		return fmt.Errorf("object %s is not a tree", treeObject.Hash)
	}

	err := os.MkdirAll(treePath, 0755)
	if err != nil {
		return err
	}

	treeEntries, err := parseTree(bufio.NewReader(bytes.NewReader(treeObject.Content)))
	if err != nil {
		return err
	}

	for _, entry := range treeEntries {
		object, err := NewObject(entry.Hash)
		if err != nil {
			return err
		}
		fullpath := path.Join(treePath, entry.Name)
		if entry.Type == ObjectTypeBlob {
			err := writeBlobToDisk(object, fullpath)
			if err != nil {
				return err
			}
		} else if entry.Type == ObjectTypeTree {
			err := writeTreeToDisk(object, fullpath)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid object type %s", entry.Type)
		}
	}

	return nil
}
