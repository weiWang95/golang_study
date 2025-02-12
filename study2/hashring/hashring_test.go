package hashring

import (
	"fmt"
	"testing"
)

func TestHashRing(t *testing.T) {
	r := NewHashRing()
	r.AddNode("node1")
	r.AddNode("node2")
	r.AddNode("node3")

	out := make(map[string]int)
	for i := 0; i < 1000; i++ {
		key := randString(i)
		node := r.Get([]byte(key))
		out[node] += 1
	}

	fmt.Println(out)

	r.RemoveNode("node2")

	out = make(map[string]int)
	for i := 0; i < 1000; i++ {
		key := randString(i)
		node := r.Get([]byte(key))
		out[node] += 1
	}

	fmt.Println(out)

	t.Fail()
}

func randString(n int) string {
	b := make([]byte, n/26+1)
	for i := range b {
		b[i] = 'a' + byte(n/26)
	}
	return string(b)
}
