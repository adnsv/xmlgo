package xg

import (
	"bytes"
	"fmt"
	"testing"
)

func TestWriter(t *testing.T) {
	WriteExample()
}

func WriteExample() {

	out := &bytes.Buffer{}
	w := NewWriter(out)

	s := "Hello, "

	w.OTag("hello")
	w.Attr("attr", 42)
	w.Attr("attr2", true)
	w.BeginContent()
	w.Write(s)
	w.Write("World!")
	w.Write(' ')
	w.Write([]string{"a", "b", "c"})
	w.CTag()

	fmt.Println(out.String())
}
