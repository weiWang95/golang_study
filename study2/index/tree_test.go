package index

import (
	"fmt"
	"testing"
)

func TestTree(t *testing.T) {
	tree, err := NewBPlusTree()
	if err != nil {
		t.Fatal(err)
	}
	defer tree.data.Close()
	defer tree.reflushInfo()

	// if err := tree.Add([]byte("a"), []byte("aaa")); err != nil {
	// 	t.Fatal(err)
	// }
	// if err := tree.Add([]byte("c"), []byte("ccc")); err != nil {
	// 	t.Fatal(err)
	// }
	// if err := tree.Add([]byte("d"), []byte("dddd")); err != nil {
	// 	t.Fatal(err)
	// }
	// if err := tree.Add([]byte("b"), []byte("bbb")); err != nil {
	// 	t.Fatal(err)
	// }

	v, err := tree.Find([]byte("a"))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("r:", string(v))
	t.Fail()
}
