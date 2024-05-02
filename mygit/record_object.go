package mygit

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type HashObjectOptions struct {
	Path  string
	Write bool
}

func HashObject(options *HashObjectOptions) error {
	blobInfo, err := HashBlob(options.Path, options.Write)
	if err != nil {
		return err
	}

	fmt.Println(blobInfo.Hash)
	return nil
}

type BlobInfo struct {
	Hash      string
	HashBytes []byte
	FileInfo  *fs.FileInfo
}

func writeObject(sha string, header []byte, content io.Reader) error {
	objectFolderPath := fmt.Sprintf(".git/objects/%s", sha[:2])
	objectPath := fmt.Sprintf(".git/objects/%s/%s", sha[:2], sha[2:])

	if _, err := os.Stat(objectPath); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(objectFolderPath, 0755); err != nil &&
			!errors.Is(err, os.ErrExist) {
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

		io.Copy(zlibWriter, content)
	}

	return nil
}

// HashBlob computes the object ID of a blob and optionally creates a blob from a file
func HashBlob(filePath string, writeOption bool) (*BlobInfo, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("unable to hash, %s is a directory", filePath)
	}

	header := fmt.Sprintf("blob %d\x00", fileInfo.Size())

	hash := sha1.New()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err = hash.Write([]byte(header)); err != nil {
		return nil, err
	}

	if _, err = io.Copy(hash, file); err != nil {
		return nil, err
	}

	shaString := fmt.Sprintf("%x", hash.Sum(nil))

	if writeOption {
		err := writeObject(shaString, []byte(header), bufio.NewReader(file))
		if err != nil {
			return nil, err
		}
	}

	return &BlobInfo{
		Hash:      shaString,
		HashBytes: hash.Sum(nil),
		FileInfo:  &fileInfo,
	}, nil
}

func HashTree(treeEntries *[]*TreeEntry, writeOption bool) ([]byte, error) {
	bufferContent := bytes.Buffer{}

	for _, entry := range *treeEntries {
		line := fmt.Sprintf("%s %s\x00", entry.Mode, entry.Name)
		bufferContent.WriteString(line)
		bufferContent.Write(entry.HashBytes)
	}

	header := fmt.Sprintf("tree %d\x00", bufferContent.Len())

	hash := sha1.New()
	if _, err := hash.Write([]byte(header)); err != nil {
		return nil, err
	}

	content := bufferContent.Bytes()
	_, err := hash.Write(content)
	if err != nil {
		return nil, err
	}

	if writeOption {
		shaString := fmt.Sprintf("%x", hash.Sum(nil))
		err := writeObject(shaString, []byte(header), bytes.NewReader(content))
		if err != nil {
			return nil, err
		}
	}

	return hash.Sum(nil), nil
}

func RecordBlob(filePath string, writeOption bool) (*TreeEntry, error) {
	blobInfo, err := HashBlob(filePath, writeOption)
	if err != nil {
		return nil, err
	}

	// Possible modes for file:
	// 100644 normal file
	// 100755 executable file
	// 120000 symbolic link
	mode := "100644"
	fileModeInfo := (*blobInfo.FileInfo).Mode()
	if fileModeInfo.IsRegular() && fileModeInfo&0111 != 0 {
		// file is executable
		mode = "100755"
	} else if fileModeInfo&fs.ModeSymlink != 0 {
		// file is a symbolic link
		mode = "120000"
	}

	return &TreeEntry{
		Mode:      mode,
		Type:      ObjectTypeBlob,
		Hash:      blobInfo.Hash,
		HashBytes: blobInfo.HashBytes,
		Name:      filepath.Base(filePath),
	}, nil
}

func RecordTree(folderPath string, writeOption bool) (*TreeEntry, error) {
	directory, err := os.Open(folderPath)
	if err != nil {
		return nil, err
	}

	fileinfo, err := directory.Stat()
	if err != nil {
		return nil, err
	}
	if !fileinfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", folderPath)
	}

	dirEntries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	entries := []*TreeEntry{}

	for _, dirEntry := range dirEntries {
		if dirEntry.Name() == ".git" {
			continue
		}
		entryPath := filepath.Join(folderPath, dirEntry.Name())
		entry, err := RecordAny(entryPath, writeOption)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	shaBytes, err := HashTree(&entries, writeOption)
	if err != nil {
		return nil, err
	}

	return &TreeEntry{
		Mode:      "40000",
		Type:      ObjectTypeTree,
		Hash:      fmt.Sprintf("%x", shaBytes),
		HashBytes: shaBytes,
		Name:      filepath.Base(folderPath),
	}, nil
}

func RecordAny(path string, writeOption bool) (*TreeEntry, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return RecordTree(path, writeOption)
	} else {
		return RecordBlob(path, writeOption)
	}
}
