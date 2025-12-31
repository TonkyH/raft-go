package raft

// NodeState Raftノードの状態を表す
type NodeState int

const (
	// Follower フォロワー状態
	Follower NodeState = iota
	// Candidate 候補者状態
	Candidate
	// Leader リーダー状態
	Leader
)

// String NodeStateの文字列表現を返す
func (s NodeState) String() string {
	switch s {
	case Follower:
		return "フォロワー"
	case Candidate:
		return "候補者"
	case Leader:
		return "リーダー"
	default:
		return "不明"
	}
}
