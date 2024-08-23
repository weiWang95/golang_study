package main

import (
	"strings"

	"github.com/sirupsen/logrus"
)

type Node struct {
	path     string
	priority uint32
	handler  func()
	children []Node
}

func NewNode(path string) *Node {
	n := new(Node)
	n.path = path
	n.priority = 1
	n.children = make([]Node, 0)
	return n
}

type Tree struct {
	root *Node
}

func NewTree() *Tree {
	t := new(Tree)
	t.root = NewNode("/")

	return t
}

func (t *Tree) AddRouter(path string, handler func()) bool {
	cur := t.root
	curPath := path

	for {
		var matched bool
		for i, item := range cur.children {
			if curPath[0] != item.path[0] {
				continue
			}

			// path: /abcdef node: /abc
			if strings.HasPrefix(curPath, item.path) {
				curPath = curPath[len(item.path):]
				cur = &cur.children[i]
				matched = true
				break
			} else {
				// path: /adc node: /abc
				// /a => [dc bc]
				var j int
				for ; j < len(item.path); j++ {
					if curPath[j] != item.path[j] {
						break
					}
				}

				// /a
				n := NewNode(item.path[0:j])
				n.priority = item.priority + 1

				// /abc => /bc
				cur.children[i].path = item.path[j:]
				n.children = append(n.children, cur.children[i])

				cur.children[i] = *n
				cur = &cur.children[i]
				curPath = curPath[j:]
				matched = true
				break
			}
		}

		if matched {
			continue
		}

		if curPath == "" {
			return false
		}

		n := NewNode(curPath)
		n.handler = handler
		cur.children = append(cur.children, *n)
		cur.priority += 1
		break
	}

	return true
}

func (t *Tree) Match(path string) (bool, func()) {
	cur := t.root
	curPath := path
	for {
		logrus.Tracef("match: path:%s", curPath)

		var matched bool
		for i, item := range cur.children {
			logrus.Tracef("compare:%s", item.path)
			if curPath[0] != item.path[0] {
				continue
			}

			if !strings.HasPrefix(curPath, item.path) {
				continue
			}

			logrus.Tracef("matched!")

			cur = &cur.children[i]
			curPath = curPath[len(item.path):]

			if curPath == "" {
				return true, item.handler
			}

			matched = true
			break
		}

		if matched {
			continue
		}
		logrus.Tracef("not match: path:%s ", curPath)

		return false, nil
	}
}
