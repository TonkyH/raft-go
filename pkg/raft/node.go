package raft

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/TonkyH/raft-go/pkg/rpc"
)

// Transport ノード間通信のインターフェース
type Transport interface {
	// RequestVote 指定ノードにRequestVote RPCを送信する
	RequestVote(nodeID int, args rpc.RequestVoteArgs) rpc.RequestVoteReply
	// AppendEntries 指定ノードにAppendEntries RPCを送信する
	AppendEntries(nodeID int, args rpc.AppendEntriesArgs) rpc.AppendEntriesReply
}

// Node Raftクラスタ内の1ノードを表す
type Node struct {
	mu sync.Mutex

	// ノード識別情報
	id        int
	peerIDs   []int
	transport Transport

	// 永続化される状態（実際の実装ではディスクに保存）
	currentTerm int
	votedFor    int
	log         *Log

	// 揮発性の状態
	commitIndex int
	lastApplied int

	// リーダーのみが持つ揮発性の状態
	nextIndex  map[int]int
	matchIndex map[int]int

	// ノードの現在状態
	state           NodeState
	config          *Config
	electionTimer   *time.Timer
	heartbeatTicker *time.Ticker

	// チャネル
	voteCh   chan bool
	appendCh chan bool
	commitCh chan rpc.LogEntry
	stopCh   chan struct{}

	// 実行状態
	running bool
}

// NewNode 新しいRaftノードを作成する
func NewNode(id int, peerIDs []int, transport Transport, config *Config) *Node {
	if config == nil {
		config = DefaultConfig()
	}
	config.Validate()

	return &Node{
		id:          id,
		peerIDs:     peerIDs,
		transport:   transport,
		currentTerm: 0,
		votedFor:    -1,
		log:         NewLog(),
		commitIndex: 0,
		lastApplied: 0,
		state:       Follower,
		config:      config,
		nextIndex:   make(map[int]int),
		matchIndex:  make(map[int]int),
		voteCh:      make(chan bool, 100),
		appendCh:    make(chan bool, 100),
		commitCh:    make(chan rpc.LogEntry, 100),
		stopCh:      make(chan struct{}),
		running:     false,
	}
}

// ID ノードのIDを返す
func (n *Node) ID() int {
	return n.id
}

// Start Raftノードを起動する
func (n *Node) Start() {
	n.mu.Lock()
	if n.running {
		n.mu.Unlock()
		return
	}
	n.running = true
	n.mu.Unlock()

	go n.run()
}

// Stop Raftノードを停止する
func (n *Node) Stop() {
	n.mu.Lock()
	if !n.running {
		n.mu.Unlock()
		return
	}
	n.running = false
	n.mu.Unlock()

	close(n.stopCh)
}

// IsRunning ノードが実行中かどうかを返す
func (n *Node) IsRunning() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.running
}

// getElectionTimeout ランダムな選挙タイムアウトを返す
func (n *Node) getElectionTimeout() time.Duration {
	min := int64(n.config.ElectionTimeoutMin)
	max := int64(n.config.ElectionTimeoutMax)
	return time.Duration(min + rand.Int63n(max-min))
}

// run Raftノードのメインループ
func (n *Node) run() {
	n.electionTimer = time.NewTimer(n.getElectionTimeout())

	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		n.mu.Lock()
		state := n.state
		n.mu.Unlock()

		switch state {
		case Follower:
			n.runFollower()
		case Candidate:
			n.runCandidate()
		case Leader:
			n.runLeader()
		}
	}
}

// runFollower フォロワー状態の処理を行う
func (n *Node) runFollower() {
	select {
	case <-n.stopCh:
		return
	case <-n.voteCh:
		n.electionTimer.Reset(n.getElectionTimeout())
	case <-n.appendCh:
		n.electionTimer.Reset(n.getElectionTimeout())
	case <-n.electionTimer.C:
		n.mu.Lock()
		log.Printf("[ノード%d] 選挙タイムアウト、候補者になります", n.id)
		n.state = Candidate
		n.mu.Unlock()
	}
}

// runCandidate 候補者状態の処理を行う
func (n *Node) runCandidate() {
	n.mu.Lock()
	n.currentTerm++
	n.votedFor = n.id
	currentTerm := n.currentTerm
	lastLogIndex, lastLogTerm := n.log.GetLastInfo()
	n.mu.Unlock()

	log.Printf("[ノード%d] 任期%dの選挙を開始", n.id, currentTerm)

	votes := 1
	votesNeeded := (len(n.peerIDs)+1)/2 + 1

	var wg sync.WaitGroup
	voteChan := make(chan bool, len(n.peerIDs))

	for _, peerID := range n.peerIDs {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			args := rpc.RequestVoteArgs{
				Term:         currentTerm,
				CandidateID:  n.id,
				LastLogIndex: lastLogIndex,
				LastLogTerm:  lastLogTerm,
			}
			reply := n.transport.RequestVote(pid, args)

			n.mu.Lock()
			if reply.Term > n.currentTerm {
				n.currentTerm = reply.Term
				n.state = Follower
				n.votedFor = -1
				n.mu.Unlock()
				return
			}
			n.mu.Unlock()

			voteChan <- reply.VoteGranted
		}(peerID)
	}

	electionTimeout := time.NewTimer(n.getElectionTimeout())
	defer electionTimeout.Stop()

	go func() {
		wg.Wait()
		close(voteChan)
	}()

	for {
		select {
		case <-n.stopCh:
			return
		case granted, ok := <-voteChan:
			if !ok {
				return
			}
			if granted {
				votes++
				if votes >= votesNeeded {
					n.mu.Lock()
					if n.state == Candidate && n.currentTerm == currentTerm {
						log.Printf("[ノード%d] 任期%dの選挙に%d票で勝利", n.id, currentTerm, votes)
						n.state = Leader
						n.initLeaderState()
					}
					n.mu.Unlock()
					return
				}
			}
		case <-electionTimeout.C:
			log.Printf("[ノード%d] 任期%dの選挙がタイムアウト", n.id, currentTerm)
			return
		case <-n.appendCh:
			n.mu.Lock()
			n.state = Follower
			n.mu.Unlock()
			return
		}
	}
}

// runLeader リーダー状態の処理を行う
func (n *Node) runLeader() {
	n.heartbeatTicker = time.NewTicker(n.config.HeartbeatInterval)
	defer n.heartbeatTicker.Stop()

	n.sendHeartbeats()

	for {
		select {
		case <-n.stopCh:
			return
		case <-n.heartbeatTicker.C:
			n.mu.Lock()
			if n.state != Leader {
				n.mu.Unlock()
				return
			}
			n.mu.Unlock()
			n.sendHeartbeats()
		}
	}
}

// initLeaderState リーダーの揮発性状態を初期化する
func (n *Node) initLeaderState() {
	lastLogIndex := n.log.LastIndex()
	for _, peerID := range n.peerIDs {
		n.nextIndex[peerID] = lastLogIndex + 1
		n.matchIndex[peerID] = 0
	}
}

// sendHeartbeats 全ピアにハートビートを送信する
func (n *Node) sendHeartbeats() {
	n.mu.Lock()
	currentTerm := n.currentTerm
	commitIndex := n.commitIndex
	n.mu.Unlock()

	for _, peerID := range n.peerIDs {
		go func(pid int) {
			n.mu.Lock()
			if n.state != Leader {
				n.mu.Unlock()
				return
			}

			prevLogIndex := n.nextIndex[pid] - 1
			prevLogTerm := 0
			if prevLogIndex > 0 {
				if entry := n.log.Get(prevLogIndex); entry != nil {
					prevLogTerm = entry.Term
				}
			}

			var entries []rpc.LogEntry
			if n.nextIndex[pid] <= n.log.Len() {
				entries = n.log.Slice(n.nextIndex[pid])
			}
			n.mu.Unlock()

			args := rpc.AppendEntriesArgs{
				Term:         currentTerm,
				LeaderID:     n.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: commitIndex,
			}

			reply := n.transport.AppendEntries(pid, args)

			n.mu.Lock()
			defer n.mu.Unlock()

			if reply.Term > n.currentTerm {
				n.currentTerm = reply.Term
				n.state = Follower
				n.votedFor = -1
				return
			}

			if n.state != Leader || n.currentTerm != currentTerm {
				return
			}

			if reply.Success {
				if len(entries) > 0 {
					n.nextIndex[pid] = entries[len(entries)-1].Index + 1
					n.matchIndex[pid] = entries[len(entries)-1].Index
					n.updateCommitIndex()
				}
			} else {
				if n.nextIndex[pid] > 1 {
					n.nextIndex[pid]--
				}
			}
		}(peerID)
	}
}

// updateCommitIndex matchIndexに基づいてcommitIndexを更新する
func (n *Node) updateCommitIndex() {
	for i := n.log.Len(); i > n.commitIndex; i-- {
		entry := n.log.Get(i)
		if entry == nil || entry.Term != n.currentTerm {
			continue
		}

		count := 1
		for _, peerID := range n.peerIDs {
			if n.matchIndex[peerID] >= i {
				count++
			}
		}

		if count > (len(n.peerIDs)+1)/2 {
			n.commitIndex = i
			log.Printf("[ノード%d] インデックス%dをコミット", n.id, i)
			break
		}
	}
}

// GetState ノードの現在状態を返す
func (n *Node) GetState() (term int, isLeader bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.currentTerm, n.state == Leader
}

// GetNodeState ノードの詳細な状態を返す
func (n *Node) GetNodeState() NodeState {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.state
}

// GetLogLength ログの長さを返す
func (n *Node) GetLogLength() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.log.Len()
}

// GetCommitIndex コミットインデックスを返す
func (n *Node) GetCommitIndex() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.commitIndex
}

// Propose 新しいコマンドをRaftクラスタに提案する
func (n *Node) Propose(command interface{}) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != Leader {
		return false
	}

	entry := rpc.LogEntry{
		Term:    n.currentTerm,
		Index:   n.log.Len() + 1,
		Command: command,
	}

	n.log.Append(entry)
	log.Printf("[ノード%d] コマンドを提案: %v, インデックス: %d", n.id, command, entry.Index)

	return true
}
