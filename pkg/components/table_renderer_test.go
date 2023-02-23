package components

import (
	"os"
	"testing"
)

func Test_CreateStreamWriter(t *testing.T) {
	w := CreateStreamWriter("table", os.Stdout)
	w.SetHeader("1", []string{"a", "b", "c"})
	w.Write([]interface{}{"aa", "bb", "cc"})
	w.Write([]interface{}{"aaa", "bbb", "ccc"})

	w.Finish()
}
