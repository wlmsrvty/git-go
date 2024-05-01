package mygit

import (
	"fmt"
)

// CatFile simply prints the content of a git object
func (o *Object) CatFile() {
	textContent := string(o.Content[:])
	fmt.Print(textContent)
}
