package index

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"sync/atomic"
)

const (
	NodeHeadLen          = 16
	NodeValueHeadLen     = 10
	LeafNodeValueHeadLen = 4
)

const (
	BitLeaf uint32 = 1 << iota
	_
	_
	_
	_
	_
	_
	BitLock
)

type Node struct {
	Value []byte
}

func NewNode(leaf bool) *Node {
	n := make([]byte, NodeCap)

	copy(n[0:NodeHeadLen], NewHead(leaf))
	return &Node{Value: n}
}

func (n *Node) Head() NodeHead {
	return n.Value[0:NodeHeadLen]
}

func (n *Node) AddKey(ptr uint64, key []byte) int {
	idx := -1
	n.Each(func(cur int, data []byte, _ uint64) bool {
		if bytes.Compare(data, key) <= 0 {
			return true
		}

		idx = cur
		return false
	})

	if idx != -1 {
		n.AddKeyAt(idx, ptr, key)
		if idx == NodeHeadLen { // 最前
			return -1
		}
		return 0 // 中间
	}

	n.AddKeyAt(int(n.Head().Used()), ptr, key)
	return 1 // 末尾
}

func (n *Node) AddKeyAt(cur int, ptr uint64, key []byte) {
	h := NewNodeValueHead(ptr, key)
	hEnd := cur + NodeValueHeadLen
	keyEnd := hEnd + len(key)
	addLen := keyEnd - cur

	used := int(n.Head().Used())
	if cur < used {
		// 后移
		copy(n.Value[keyEnd:used+addLen], n.Value[cur:used])
	}

	copy(n.Value[cur:hEnd], h)
	copy(n.Value[hEnd:keyEnd], key)

	n.Head().SetUsed(uint16(used + addLen))
	n.Head().SetCount(n.Head().Count() + 1)
}

func (n *Node) AddLeafValue(key, value []byte) int {
	idx := -1

	n.EachLeaf(func(cur int, k, v []byte) bool {
		if bytes.Compare(k, key) <= 0 {
			return true
		}

		idx = cur
		return false
	})

	if idx != -1 {
		n.AddLeafValueAt(idx, key, value)
		if idx == NodeHeadLen { // 最前
			return -1
		}
		return 0 // 中间
	}

	n.AddLeafValueAt(int(n.Head().Used()), key, value)
	return 1 // 末尾
}

func (n *Node) AddLeafValueAt(cur int, key, value []byte) {
	h := NewLeafValueHead(key, value)
	hEnd := cur + LeafNodeValueHeadLen
	keyEnd := hEnd + len(key)
	valueEnd := keyEnd + len(value)
	addLen := valueEnd - cur

	used := int(n.Head().Used())
	if cur < used {
		// 后移
		copy(n.Value[valueEnd:used+addLen], n.Value[cur:used])
	}

	copy(n.Value[cur:hEnd], h)
	copy(n.Value[hEnd:keyEnd], key)
	copy(n.Value[keyEnd:valueEnd], value)

	n.Head().SetUsed(uint16(used + addLen))
	n.Head().SetCount(n.Head().Count() + 1)
}

func (n *Node) Each(fn func(cur int, data []byte, ptr uint64) bool) {
	head := n.Head()
	used := int(head.Used())

	if used == 0 || used == NodeHeadLen {
		return
	}

	cur := NodeHeadLen

	var ptr uint64
	for {
		if cur >= used {
			break
		}

		headEnd := cur + NodeValueHeadLen
		vHead := NodeValueHead(n.Value[cur:headEnd])
		ptr = vHead.Ptr()

		valueEnd := headEnd + int(vHead.KeyLen())
		key := n.Value[headEnd:valueEnd]

		if !fn(cur, key, ptr) {
			break
		}

		cur = valueEnd
	}

}

func (n *Node) First() (ptr uint64, key []byte) {
	headEnd := NodeHeadLen + NodeValueHeadLen
	h := NodeValueHead(n.Value[NodeHeadLen:headEnd])

	keyEnd := headEnd + int(h.KeyLen())
	key = n.Value[headEnd:keyEnd]
	ptr = h.Ptr()
	return
}

func (n *Node) EachLeaf(fn func(cur int, key, data []byte) bool) {
	head := n.Head()
	used := int(head.Used())

	cur := NodeHeadLen
	for {
		if cur >= used {
			break
		}

		headEnd := cur + LeafNodeValueHeadLen
		vHead := LeafValueHead(n.Value[cur : cur+LeafNodeValueHeadLen])

		keyEnd := headEnd + int(vHead.KeyLen())
		bodyEnd := keyEnd + int(vHead.BodyLen())

		key := n.Value[headEnd:keyEnd]
		data := n.Value[keyEnd:bodyEnd]

		if !fn(cur, key, data) {
			break
		}

		cur = bodyEnd
	}
}

func (n *Node) Enough(size uint16) bool {
	head := n.Head()
	return uint16(NodeCap)-head.Used() >= size
}

func (n *Node) Find(key []byte) (cur int, ptr uint64, k []byte) {
	n.Each(func(idx int, data []byte, p uint64) bool {
		cur, ptr, k = idx, p, data
		return bytes.Compare(data, key) < 0
	})
	return
}

func (n *Node) FindValue(key []byte) (cur int, rk []byte, data []byte) {
	n.EachLeaf(func(c int, k, d []byte) bool {
		cur, rk, data = c, k, d
		return !bytes.Equal(key, k)
	})
	return
}

func (n *Node) Update(cur int, ptr uint64, key []byte) {
	used := int(n.Head().Used())

	oldHead := NodeValueHead(n.Value[cur : cur+NodeValueHeadLen])
	oldKeyLen := int(oldHead.KeyLen())
	oldEnd := cur + NodeValueHeadLen + oldKeyLen

	kenLen := len(key)
	headEnd := cur + NodeValueHeadLen
	newEnd := headEnd + kenLen

	newUsed := used + kenLen - oldKeyLen

	// TODO: 存不下的情况,需要分裂
	// 调整后面的数据位置
	copy(n.Value[newEnd:newUsed], n.Value[oldEnd:used])

	h := NewNodeValueHead(ptr, key)
	copy(n.Value[cur:headEnd], h)
	copy(n.Value[headEnd:newEnd], key)

	n.Head().SetUsed(uint16(newUsed))
}

func (n *Node) Split() *Node {
	h := n.Head()
	used := h.Used()
	count := h.Count()
	half := int(h.Count() / 2)

	var cur int
	var num int
	n.Each(func(idx int, data []byte, ptr uint64) bool {
		num += 1
		cur = idx

		return num < half
	})

	newNode := NewNode(false)
	newUsed := NodeHeadLen + used - uint16(cur)
	copy(newNode.Value[NodeHeadLen:newUsed], n.Value[cur:used])

	newNode.Head().SetUsed(newUsed)
	newNode.Head().SetCount(count - uint16(half))

	n.Head().SetUsed(uint16(cur))
	n.Head().SetCount(uint16(half))

	return newNode
}

func (n *Node) SplitLeaf() *Node {
	h := n.Head()
	used := h.Used()
	count := h.Count()
	half := int(h.Count() / 2)

	var cur int
	var num int
	n.EachLeaf(func(idx int, key, data []byte) bool {
		num += 1
		cur = idx

		return num < half
	})

	newNode := NewNode(true)
	newUsed := NodeHeadLen + used - uint16(cur)
	copy(newNode.Value[NodeHeadLen:newUsed], n.Value[cur:used])

	newNode.Head().SetUsed(newUsed)
	newNode.Head().SetCount(count - uint16(half))

	n.Head().SetUsed(uint16(cur))
	n.Head().SetCount(uint16(half))

	return newNode
}

func (n *Node) String() string {
	var buf strings.Builder
	h := NodeHead(n.Value[0:NodeHeadLen])
	buf.WriteString(h.String())

	if n.Head().IsLeaf() {
		n.EachLeaf(func(cur int, key, data []byte) bool {
			buf.WriteString(fmt.Sprintf(" %s:%s ", string(key), string(data)))
			return true
		})
	} else {
		n.Each(func(cur int, data []byte, ptr uint64) bool {
			buf.WriteString(fmt.Sprintf(" %s:%d ", string(data), ptr))
			return true
		})
	}

	return buf.String()
}

type NodeHead []byte

func NewHead(leaf bool) NodeHead {
	h := make(NodeHead, NodeHeadLen)

	var bit uint32
	if leaf {
		bit = bit | BitLeaf
	}
	binary.BigEndian.PutUint32(h[0:4], bit)
	h.SetUsed(NodeHeadLen)
	return h
}

func (n NodeHead) Bit() uint32 {
	return binary.BigEndian.Uint32(n[0:4])
}

func (n NodeHead) IsLeaf() bool {
	return n.Bit()&BitLeaf == 1
}

func (n NodeHead) Locked() bool {
	return n.Bit()&BitLock == 1
}

func (n NodeHead) Lock() bool {
	bit := n.Bit()
	return atomic.CompareAndSwapUint32(&bit, bit, bit|BitLock)
}

func (n NodeHead) Unlock() bool {
	bit := n.Bit()
	return atomic.CompareAndSwapUint32(&bit, bit, bit&^BitLock)
}

func (n NodeHead) Next() uint64 {
	return binary.BigEndian.Uint64(n[4:12])
}

func (n NodeHead) SetNext(ptr uint64) {
	binary.BigEndian.PutUint64(n[4:12], ptr)
}

func (n NodeHead) Count() uint16 {
	return binary.BigEndian.Uint16(n[12:14])
}

func (n NodeHead) SetCount(count uint16) {
	binary.BigEndian.PutUint16(n[12:14], count)
}

func (n NodeHead) Used() uint16 {
	return binary.BigEndian.Uint16(n[14:16])
}

func (n NodeHead) SetUsed(used uint16) {
	binary.BigEndian.PutUint16(n[14:16], used)
}

func (n NodeHead) String() string {
	return fmt.Sprintf(
		"b:%b,p:%d,c:%d,u:%d",
		binary.BigEndian.Uint32(n[0:4]),
		n.Next(), n.Count(), n.Used(),
	)
}

type NodeValueHead []byte

func NewNodeValueHead(ptr uint64, key []byte) NodeValueHead {
	h := make(NodeValueHead, NodeValueHeadLen)

	binary.BigEndian.PutUint16(h[0:2], uint16(len(key)))
	binary.BigEndian.PutUint64(h[2:10], ptr)

	return h
}

func (n NodeValueHead) KeyLen() uint16 {
	return binary.BigEndian.Uint16(n[0:2])
}

func (n NodeValueHead) Ptr() uint64 {
	return binary.BigEndian.Uint64(n[2:10])
}

type LeafValueHead []byte

func NewLeafValueHead(key, value []byte) LeafValueHead {
	h := make(LeafValueHead, LeafNodeValueHeadLen)

	binary.BigEndian.PutUint16(h[0:2], uint16(len(key)))
	binary.BigEndian.PutUint16(h[2:4], uint16(len(value)))

	return h
}

func (n LeafValueHead) KeyLen() uint16 {
	return binary.BigEndian.Uint16(n[0:2])
}

func (n LeafValueHead) BodyLen() uint16 {
	return binary.BigEndian.Uint16(n[2:4])
}
