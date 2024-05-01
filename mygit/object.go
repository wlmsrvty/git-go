package mygit

import (
	"bufio"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Object represents a git object
type Object struct {
	Type    ObjectType
	Size    int
	Path    string
	Content []byte
	Hash    string
}

// Returns the path to the object file in the .git directory
func getObjectPath(sha string) string {
	return fmt.Sprintf(".git/objects/%s/%s", sha[:2], sha[2:])
}

// Creates a new git object from the given hash of the object
func NewObject(sha string) (*Object, error) {
	path := getObjectPath(sha)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("object %s does not exist", sha)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer zlibReader.Close()

	bufioReader := bufio.NewReader(zlibReader)

	objType, objSize, err := parseHeader(bufioReader)
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(bufioReader)
	if err != nil {
		return nil, err
	}

	return &Object{
		Type:    objType,
		Size:    objSize,
		Path:    path,
		Content: content,
		Hash:    sha,
	}, nil
}

// parseHeader parses the header of a git object
// Git object header format: <type> <size>\x00<content>
func parseHeader(objectReader *bufio.Reader) (ObjectType, int, error) {

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
