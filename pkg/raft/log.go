package raft

import "github.com/TonkyH/raft-go/pkg/rpc"

// Log Raftのログを管理する構造体
type Log struct {
	entries []rpc.LogEntry
}

// NewLog 新しいLogを作成する
func NewLog() *Log {
	return &Log{
		entries: make([]rpc.LogEntry, 0),
	}
}

// Append ログにエントリを追加する
func (l *Log) Append(entry rpc.LogEntry) {
	l.entries = append(l.entries, entry)
}

// Get 指定インデックスのエントリを取得する（1-indexed）
func (l *Log) Get(index int) *rpc.LogEntry {
	if index <= 0 || index > len(l.entries) {
		return nil
	}
	return &l.entries[index-1]
}

// LastIndex 最後のログエントリのインデックスを返す
func (l *Log) LastIndex() int {
	return len(l.entries)
}

// LastTerm 最後のログエントリの任期を返す
func (l *Log) LastTerm() int {
	if len(l.entries) == 0 {
		return 0
	}
	return l.entries[len(l.entries)-1].Term
}

// GetLastInfo 最後のログエントリのインデックスと任期を返す
func (l *Log) GetLastInfo() (index int, term int) {
	if len(l.entries) == 0 {
		return 0, 0
	}
	lastEntry := l.entries[len(l.entries)-1]
	return lastEntry.Index, lastEntry.Term
}

// Slice 指定範囲のエントリを返す（fromIndex以降、1-indexed）
func (l *Log) Slice(fromIndex int) []rpc.LogEntry {
	if fromIndex <= 0 || fromIndex > len(l.entries) {
		return nil
	}
	// コピーを返す（元のスライスを保護）
	result := make([]rpc.LogEntry, len(l.entries)-(fromIndex-1))
	copy(result, l.entries[fromIndex-1:])
	return result
}

// TruncateAfter 指定インデックス以降のエントリを削除する（1-indexed）
func (l *Log) TruncateAfter(index int) {
	if index < 0 {
		index = 0
	}
	if index < len(l.entries) {
		l.entries = l.entries[:index]
	}
}

// Len ログの長さを返す
func (l *Log) Len() int {
	return len(l.entries)
}

// Entries 全エントリのコピーを返す
func (l *Log) Entries() []rpc.LogEntry {
	result := make([]rpc.LogEntry, len(l.entries))
	copy(result, l.entries)
	return result
}
