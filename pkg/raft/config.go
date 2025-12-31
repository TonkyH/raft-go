package raft

import "time"

// Config Raftノードの設定を保持する
type Config struct {
	// ElectionTimeoutMin 選挙タイムアウトの最小値
	ElectionTimeoutMin time.Duration
	// ElectionTimeoutMax 選挙タイムアウトの最大値
	ElectionTimeoutMax time.Duration
	// HeartbeatInterval ハートビートの送信間隔
	HeartbeatInterval time.Duration
}

// DefaultConfig デフォルトの設定を返す
func DefaultConfig() *Config {
	return &Config{
		ElectionTimeoutMin: 150 * time.Millisecond,
		ElectionTimeoutMax: 300 * time.Millisecond,
		HeartbeatInterval:  50 * time.Millisecond,
	}
}

// Validate 設定値を検証する
func (c *Config) Validate() error {
	if c.ElectionTimeoutMin <= 0 {
		c.ElectionTimeoutMin = 150 * time.Millisecond
	}
	if c.ElectionTimeoutMax <= c.ElectionTimeoutMin {
		c.ElectionTimeoutMax = c.ElectionTimeoutMin + 150*time.Millisecond
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = 50 * time.Millisecond
	}
	// ハートビートは選挙タイムアウトより短くなければならない
	if c.HeartbeatInterval >= c.ElectionTimeoutMin {
		c.HeartbeatInterval = c.ElectionTimeoutMin / 3
	}
	return nil
}
