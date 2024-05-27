package mygit

import (
	"fmt"
	"os"
)

func writeBlobToDisk(o *Object, fullPath string) error {
	if o.Type != ObjectTypeBlob {
		return fmt.Errorf("object %s is not a blob", o.Hash)
	}
	err := os.WriteFile(fullPath, o.Content, 0644)
	if err != nil {
		return err
	}
	return nil
}
