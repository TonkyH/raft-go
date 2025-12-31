package rpc

// LogEntry Raftログの1エントリを表す
type LogEntry struct {
	// Term このエントリが作成された任期
	Term int
	// Index ログ内のインデックス（1から開始）
	Index int
	// Command 実行するコマンド
	Command interface{}
}

// RequestVoteArgs RequestVote RPCの引数
type RequestVoteArgs struct {
	// Term 候補者の任期
	Term int
	// CandidateID 投票を求める候補者のID
	CandidateID int
	// LastLogIndex 候補者の最後のログエントリのインデックス
	LastLogIndex int
	// LastLogTerm 候補者の最後のログエントリの任期
	LastLogTerm int
}

// RequestVoteReply RequestVote RPCの応答
type RequestVoteReply struct {
	// Term 応答者の現在の任期（候補者が更新用に使用）
	Term int
	// VoteGranted 投票が承認されたかどうか
	VoteGranted bool
}

// AppendEntriesArgs AppendEntries RPCの引数
type AppendEntriesArgs struct {
	// Term リーダーの任期
	Term int
	// LeaderID リーダーのID
	LeaderID int
	// PrevLogIndex 新エントリの直前のログインデックス
	PrevLogIndex int
	// PrevLogTerm PrevLogIndexの任期
	PrevLogTerm int
	// Entries 保存するログエントリ（ハートビート時は空）
	Entries []LogEntry
	// LeaderCommit リーダーのcommitIndex
	LeaderCommit int
}

// AppendEntriesReply AppendEntries RPCの応答
type AppendEntriesReply struct {
	// Term 応答者の現在の任期
	Term int
	// Success PrevLogIndex/PrevLogTermが一致した場合true
	Success bool
}
