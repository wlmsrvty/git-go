package mygit

import (
	"fmt"
)

func (o *Object) CatFile() {
	textContent := string(o.Content[:])
	fmt.Print(textContent)
}
