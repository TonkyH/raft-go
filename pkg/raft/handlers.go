package raft

import (
	"log"

	"github.com/TonkyH/raft-go/pkg/rpc"
)

// HandleRequestVote RequestVote RPCを処理する
func (n *Node) HandleRequestVote(args rpc.RequestVoteArgs) rpc.RequestVoteReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := rpc.RequestVoteReply{
		Term:        n.currentTerm,
		VoteGranted: false,
	}

	// 古い任期からのリクエストは拒否
	if args.Term < n.currentTerm {
		return reply
	}

	// より新しい任期を発見したら更新してフォロワーに戻る
	if args.Term > n.currentTerm {
		n.currentTerm = args.Term
		n.state = Follower
		n.votedFor = -1
	}

	reply.Term = n.currentTerm

	// 投票可能かチェック
	// 候補者のログが自分と同じか新しい場合のみ投票
	lastLogIndex, lastLogTerm := n.log.GetLastInfo()
	logOk := args.LastLogTerm > lastLogTerm ||
		(args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogIndex)

	if (n.votedFor == -1 || n.votedFor == args.CandidateID) && logOk {
		n.votedFor = args.CandidateID
		reply.VoteGranted = true
		log.Printf("[ノード%d] 任期%dでノード%dに投票", n.id, args.Term, args.CandidateID)

		// 投票リクエスト受信を通知
		select {
		case n.voteCh <- true:
		default:
		}
	}

	return reply
}

// HandleAppendEntries AppendEntries RPCを処理する
func (n *Node) HandleAppendEntries(args rpc.AppendEntriesArgs) rpc.AppendEntriesReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := rpc.AppendEntriesReply{
		Term:    n.currentTerm,
		Success: false,
	}

	// 古い任期からのリクエストは拒否
	if args.Term < n.currentTerm {
		return reply
	}

	// より新しい任期を発見したら更新
	if args.Term > n.currentTerm {
		n.currentTerm = args.Term
		n.votedFor = -1
	}

	n.state = Follower
	reply.Term = n.currentTerm

	// AppendEntries受信を通知（選挙タイマーリセット用）
	select {
	case n.appendCh <- true:
	default:
	}

	// ログの一貫性チェック
	if args.PrevLogIndex > 0 {
		if args.PrevLogIndex > n.log.Len() {
			return reply
		}
		entry := n.log.Get(args.PrevLogIndex)
		if entry == nil || entry.Term != args.PrevLogTerm {
			// 不一致エントリを削除
			n.log.TruncateAfter(args.PrevLogIndex - 1)
			return reply
		}
	}

	// 新エントリを追加
	for i, entry := range args.Entries {
		index := args.PrevLogIndex + i + 1
		existingEntry := n.log.Get(index)
		if existingEntry != nil {
			if existingEntry.Term != entry.Term {
				n.log.TruncateAfter(index - 1)
				n.log.Append(entry)
			}
		} else {
			n.log.Append(entry)
		}
	}

	// commitIndexを更新
	if args.LeaderCommit > n.commitIndex {
		lastNewEntry := args.PrevLogIndex + len(args.Entries)
		if lastNewEntry > 0 {
			if args.LeaderCommit < lastNewEntry {
				n.commitIndex = args.LeaderCommit
			} else {
				n.commitIndex = lastNewEntry
			}
		}
	}

	reply.Success = true

	if len(args.Entries) > 0 {
		log.Printf("[ノード%d] %d個のエントリを追加、ログ長: %d", n.id, len(args.Entries), n.log.Len())
	}

	return reply
}
