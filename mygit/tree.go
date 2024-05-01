package mygit

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

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

type PrintTreeContentOptions struct {
	NameOnly bool
}

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

func parseTree(objContent *bufio.Reader) ([]TreeEntry, error) {
	// Format of the tree object:
	// 	tree <size>\x00<entry><entry>...
	// Format of the entry:
	//	<mode> <name>\x00<20_byte_hash>

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
