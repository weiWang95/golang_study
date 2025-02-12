package index

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"golang_study/study2/lru"
)

const NodeCap int = 1024
const MaxNodeValueCount = 2

type BPlusTree struct {
	lru  *lru.LRU[uint64, *Node]
	root *Node

	data *Data
	info IndexInfo
}

func NewBPlusTree() (*BPlusTree, error) {
	t := new(BPlusTree)

	data, err := NewData("index.db")
	if err != nil {
		return nil, err
	}
	t.data = data

	t.lru = lru.NewLRU[uint64, *Node](10)

	if err := t.loadInfo(); err != nil {
		return nil, err
	}
	if err := t.loadRoot(); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *BPlusTree) Add(key []byte, value []byte) error {
	n := t.root
	if n.Head().Count() == 0 {
		leaf := NewNode(true)
		leaf.AddLeafValue(key, value)

		fmt.Println(leaf)

		ptr := t.info.Total()
		t.data.WriteAt(leaf.Value, int64(ptr))
		t.info.IncTotal(uint64(NodeCap))

		n.AddKey(ptr, key)
		fmt.Println(n)
		return t.data.WriteAt(n.Value, IndexInfoLen)
	}

	cur, p, _ := n.Find(key)
	nn, err := t.loadNode(p)
	if err != nil {
		return err
	}

	return t.add(n, cur, p, nn, key, value)
}

func (t *BPlusTree) add(parent *Node, nodeCur int, nptr uint64, n *Node, key []byte, value []byte) error {
	h := n.Head()
	if !h.IsLeaf() {
		if h.Count() == 0 {
			leaf := NewNode(true)
			leaf.AddLeafValue(key, value)

			ptr := t.info.Total()
			t.data.WriteAt(leaf.Value, int64(ptr))
			t.info.IncTotal(uint64(NodeCap))

			n.AddKey(ptr, key)
			return t.data.WriteAt(n.Value, int64(nptr))
		}

		if h.Count()+1 >= MaxNodeValueCount {
			newNode := n.Split()
			newPtr := t.info.Total()

			_, k := newNode.First()
			parent.AddKey(newPtr, k) // TODO: 链式分裂

			t.data.WriteAt(newNode.Value, int64(newPtr))
			t.info.IncTotal(uint64(NodeCap))

			if bytes.Compare(key, k) == 1 {
				n = newNode
			}

			// 叶子节点
			// newNode.Head().SetNext(n.Head().Next())
			// n.Head().SetNext(newPtr)
		}

		cur, p, k := n.Find(key)

		// 插入末尾
		if bytes.Compare(k, key) < 0 {
			n.Update(cur, p, key)
			t.data.WriteAt(n.Value, int64(nptr))
		}

		nn, err := t.loadNode(p)
		if err != nil {
			return err
		}

		return t.add(n, cur, p, nn, key, value)
	}

	if h.Count()+1 >= MaxNodeValueCount {
		newNode := n.Split()
		newPtr := t.info.Total()

		newNode.Head().SetNext(n.Head().Next())
		n.Head().SetNext(newPtr)

		_, k := newNode.First()
		parent.AddKey(newPtr, k) // TODO: 链式分裂

		t.data.WriteAt(newNode.Value, int64(newPtr))
		t.info.IncTotal(uint64(NodeCap))

		if bytes.Compare(key, k) == 1 {
			t.data.WriteAt(n.Value, int64(nptr))
			n = newNode
		}
	}

	n.AddLeafValue(key, value)
	return t.data.WriteAt(n.Value, int64(nptr))
}

func (t *BPlusTree) Find(key []byte) ([]byte, error) {
	n := t.root
	log.Println(n.String())

	_, ptr, _ := n.Find(key)
	if ptr == 0 {
		return nil, nil
	}

	nn, err := t.loadNode(ptr)
	if err != nil {
		return nil, err
	}

	return t.find(nn, key)
}

func (t *BPlusTree) find(n *Node, key []byte) ([]byte, error) {
	log.Println(n.String())

	if !n.Head().IsLeaf() {
		_, ptr, _ := n.Find(key)
		nn, err := t.loadNode(ptr)
		if err != nil {
			return nil, err
		}

		return t.find(nn, key)
	}

	_, k, v := n.FindValue(key)
	if !bytes.Equal(k, key) {
		return nil, nil
	}

	return v, nil
}

func (t *BPlusTree) loadNode(ptr uint64) (*Node, error) {
	r := t.lru.Get(ptr)
	if r != nil {
		return *r, nil
	}

	n, err := t.loadNodeFromDisk(ptr)
	if err != nil {
		return nil, err
	}
	t.lru.Add(ptr, n)
	return n, nil
}

func (t *BPlusTree) loadNodeFromDisk(ptr uint64) (*Node, error) {
	bs, err := t.data.ReadNodeAt(int64(ptr))
	if err != nil {
		return nil, err
	}
	if len(bs) != NodeCap {
		return nil, fmt.Errorf("invalid data")
	}
	return &Node{Value: bs}, nil
}

func (t *BPlusTree) loadRoot() error {
	val, err := t.data.ReadNodeAt(IndexInfoLen)
	if err != nil && err != io.EOF {
		return err
	}
	if len(val) != 0 {
		t.root = &Node{Value: val}
		return nil
	}

	t.root = NewNode(false)
	fmt.Println("root:", t.root)
	t.info.IncTotal(uint64(NodeCap))
	return t.data.WriteAt(t.root.Value, IndexInfoLen)
}

func (t *BPlusTree) loadInfo() error {
	info := NewIndexInfo()
	_, err := t.data.ReadAt(info, 0)
	if err == nil {
		t.info = info
		return nil
	}

	info.SetTotal(IndexInfoLen)
	t.info = info
	return t.reflushInfo()
}

func (t *BPlusTree) reflushInfo() error {
	return t.data.WriteAt(t.info, 0)
}
