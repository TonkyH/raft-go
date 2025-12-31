package rpc

import (
	"sync"
)

// NodeInterface Raftノードのインターフェース
type NodeInterface interface {
	ID() int
	IsRunning() bool
	HandleRequestVote(args RequestVoteArgs) RequestVoteReply
	HandleAppendEntries(args AppendEntriesArgs) AppendEntriesReply
}

// InMemoryTransport テスト用のインメモリ通信実装
type InMemoryTransport struct {
	mu    sync.RWMutex
	nodes map[int]NodeInterface
}

func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{
		nodes: make(map[int]NodeInterface),
	}
}

// RegisterNode ノードを登録する
func (t *InMemoryTransport) RegisterNode(node NodeInterface) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes[node.ID()] = node
}

// UnregisterNode ノードの登録を解除する
func (t *InMemoryTransport) UnregisterNode(nodeID int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.nodes, nodeID)
}

// RequestVote 指定ノードにRequestVote RPCを送信する
func (t *InMemoryTransport) RequestVote(nodeID int, args RequestVoteArgs) RequestVoteReply {
	t.mu.RLock()
	node, ok := t.nodes[nodeID]
	t.mu.RUnlock()

	if !ok || !node.IsRunning() {
		return RequestVoteReply{Term: 0, VoteGranted: false}
	}

	return node.HandleRequestVote(args)
}

// AppendEntries 指定ノードにAppendEntries RPCを送信する
func (t *InMemoryTransport) AppendEntries(nodeID int, args AppendEntriesArgs) AppendEntriesReply {
	t.mu.RLock()
	node, ok := t.nodes[nodeID]
	t.mu.RUnlock()

	if !ok || !node.IsRunning() {
		return AppendEntriesReply{Term: 0, Success: false}
	}

	return node.HandleAppendEntries(args)
}
