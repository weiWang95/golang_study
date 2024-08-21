package main

import "context"

type NodeState uint8

const (
	StateFollower = iota
	StateCandidate
	StateLeader
)

type HearthbeatMsg struct {
	NodeId uint64
	Iterm  uint64
}

type Node struct {
	id    uint64
	state NodeState
	iterm uint64

	leaderNodeId uint64
}

func (n *Node) Run(ctx context.Context) error {
	return nil
}
