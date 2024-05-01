package mygit

import (
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
)

type HashObjectOptions struct {
	Path  string
	Write bool
}

// HashObject computes the object ID and optionally creates a blob from a file
func HashObject(options *HashObjectOptions) error {
	fileInfo, err := os.Stat(options.Path)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("unable to hash, %s is a directory", options.Path)
	}

	header := fmt.Sprintf("blob %d\000", fileInfo.Size())

	hash := sha1.New()

	file, err := os.Open(options.Path)
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

	if !options.Write {
		return nil
	}

	objectFolderPath := fmt.Sprintf(".git/objects/%s", shaString[:2])
	objectPath := fmt.Sprintf(".git/objects/%s/%s", shaString[:2], shaString[2:])

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

		file.Seek(0, 0)
		io.Copy(zlibWriter, file)
	}

	return nil
}
