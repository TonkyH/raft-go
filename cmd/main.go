package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/TonkyH/raft-go/pkg/raft"
	"github.com/TonkyH/raft-go/pkg/rpc"
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Println("===========================================")
	fmt.Println("    Raft 合意アルゴリズム デモ")
	fmt.Println("===========================================")
	fmt.Println()

	// トランスポート層を作成
	transport := rpc.NewInMemoryTransport()

	// クラスタを作成（5ノード）
	numNodes := 5
	config := raft.DefaultConfig()
	cluster := raft.NewCluster(numNodes, transport, config)

	// 各ノードをトランスポートに登録
	for _, node := range cluster.GetNodes() {
		transport.RegisterNode(node)
	}

	// クラスタを起動
	fmt.Printf("Raftクラスタを起動中（%dノード）...\n", numNodes)
	cluster.Start()

	// リーダー選出を待つ
	fmt.Println("リーダー選出を待機中...")
	time.Sleep(2 * time.Second)

	// 現在の状態を表示
	fmt.Println("\n--- 初期状態 ---")
	cluster.PrintStatus()

	// リーダーを見つけてコマンドを提案
	fmt.Println("\n--- コマンド提案 ---")
	for i := 0; i < 5; i++ {
		leader := cluster.GetLeader()
		if leader != nil {
			command := fmt.Sprintf("SET key%d value%d", i, i)
			if leader.Propose(command) {
				fmt.Printf("[デモ] ノード%dにコマンドを提案: %s\n", leader.ID(), command)
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	// 最終状態を表示
	fmt.Println("\n--- コマンド提案後の状態 ---")
	cluster.PrintStatus()

	// リーダー障害をシミュレート
	fmt.Println("\n--- リーダー障害シミュレーション ---")
	leader := cluster.GetLeader()
	if leader != nil {
		fmt.Printf("リーダー（ノード%d）を停止します...\n", leader.ID())
		leader.Stop()
		transport.UnregisterNode(leader.ID())
	}

	// 新リーダー選出を待つ
	fmt.Println("新リーダー選出を待機中...")
	time.Sleep(2 * time.Second)

	// 障害後の状態を表示
	fmt.Println("\n--- リーダー障害後の状態 ---")
	cluster.PrintStatus()

	// 新リーダーでコマンド提案
	fmt.Println("\n--- 新リーダーでのコマンド提案 ---")
	newLeader := cluster.GetLeader()
	if newLeader != nil {
		command := "SET recovery_key recovery_value"
		if newLeader.Propose(command) {
			fmt.Printf("[デモ] 新リーダー（ノード%d）にコマンドを提案: %s\n", newLeader.ID(), command)
		}
	}

	time.Sleep(500 * time.Millisecond)

	// 最終状態を表示
	fmt.Println("\n--- 最終状態 ---")
	cluster.PrintStatus()

	// クラスタを停止
	cluster.Stop()

	fmt.Println("\n===========================================")
	fmt.Println("    Raftデモ完了")
	fmt.Println("===========================================")
}
