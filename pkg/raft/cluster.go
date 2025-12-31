package raft

import (
	"fmt"
	"sync"
)

// Cluster Raftクラスタ全体を管理する
type Cluster struct {
	mu        sync.RWMutex
	nodes     []*Node
	transport Transport
	config    *Config
}

// NewCluster 新しいクラスタを作成する
func NewCluster(numNodes int, transport Transport, config *Config) *Cluster {
	if config == nil {
		config = DefaultConfig()
	}

	cluster := &Cluster{
		nodes:     make([]*Node, numNodes),
		transport: transport,
		config:    config,
	}

	// ピアIDのリストを作成
	for i := 0; i < numNodes; i++ {
		peerIDs := make([]int, 0, numNodes-1)
		for j := 0; j < numNodes; j++ {
			if i != j {
				peerIDs = append(peerIDs, j)
			}
		}
		cluster.nodes[i] = NewNode(i, peerIDs, transport, config)
	}

	return cluster
}

// GetNode 指定IDのノードを返す
func (c *Cluster) GetNode(id int) *Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if id < 0 || id >= len(c.nodes) {
		return nil
	}
	return c.nodes[id]
}

// GetNodes 全ノードを返す
func (c *Cluster) GetNodes() []*Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Node, len(c.nodes))
	copy(result, c.nodes)
	return result
}

// Start クラスタ内の全ノードを起動する
func (c *Cluster) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, node := range c.nodes {
		node.Start()
	}
}

// Stop クラスタ内の全ノードを停止する
func (c *Cluster) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, node := range c.nodes {
		node.Stop()
	}
}

// GetLeader 現在のリーダーノードを返す（いない場合はnil）
func (c *Cluster) GetLeader() *Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, node := range c.nodes {
		if _, isLeader := node.GetState(); isLeader {
			return node
		}
	}
	return nil
}

// PrintStatus クラスタの状態を表示する
func (c *Cluster) PrintStatus() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, node := range c.nodes {
		term, isLeader := node.GetState()
		state := node.GetNodeState()
		logLen := node.GetLogLength()
		commitIdx := node.GetCommitIndex()

		status := "実行中"
		if !node.IsRunning() {
			status = "停止中"
		}

		leaderMark := ""
		if isLeader {
			leaderMark = " ★"
		}

		fmt.Printf("ノード%d: 状態=%s, 任期=%d, ログ長=%d, コミット済み=%d [%s]%s\n",
			node.ID(), state, term, logLen, commitIdx, status, leaderMark)
	}
}
