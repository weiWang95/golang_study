package hashring

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
)

type HashRing struct {
	nodes map[uint32]string
	ring  []uint32

	copies uint8
}

func NewHashRing() *HashRing {
	r := new(HashRing)
	r.nodes = make(map[uint32]string)
	r.ring = make([]uint32, 0)
	r.copies = 32
	return r
}

func (r *HashRing) AddNode(name string) {
	for i := uint8(0); i < r.copies; i++ {
		hash := r.hashFun([]byte(fmt.Sprintf("%s#%d", name, i)))
		r.nodes[hash] = name
		r.ring = append(r.ring, hash)
	}

	sort.Slice(r.ring, func(i, j int) bool {
		return r.ring[i] < r.ring[j]
	})
}

func (r *HashRing) RemoveNode(name string) {
	for i := uint8(0); i < r.copies; i++ {
		hash := r.hashFun([]byte(fmt.Sprintf("%s#%d", name, i)))
		delete(r.nodes, hash)
		for j := 0; j < len(r.ring); j++ {
			if r.ring[j] == hash {
				r.ring = append(r.ring[:j], r.ring[j+1:]...)
			}
		}
	}
}

func (r *HashRing) Get(key []byte) string {
	hash := r.hashFun(key)

	for i := 0; i < len(r.ring); i++ {
		if hash < r.ring[i] {
			return r.nodes[r.ring[i]]
		}
	}

	return r.nodes[r.ring[0]]
}

func (r *HashRing) hashFun(key []byte) uint32 {
	s := sha256.New()
	s.Write(key)
	b := s.Sum(nil)

	return binary.BigEndian.Uint32(b)
}
