package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func main() {
	// инициалзация схемы из md
	var table = [][]uint64{
		{0, 1, 3, 7},
		{1, 0, 1, 0},
		{3, 1, 0, 2},
		{7, 0, 2, 1},
	}

	manager := NewManager()
	manager.Init(table)
	fmt.Println("\n== Таблицы маршрутизации после инициализации ==")
	manager.ShowTables()
	manager.Start()

	time.Sleep(2 * time.Second)

	//fmt.Println("\n== Изменение посел ожидания ==")
	manager.UpdateConnectCost(0, 1, 6)
	//
	time.Sleep(4 * time.Second)

	manager.ShowTables()

	manager.Stop()

}

type NodeManager struct {
	nodes map[uint64]*Node
	stop  chan struct{}
	wg    sync.WaitGroup
}

func NewManager() *NodeManager {
	return &NodeManager{
		nodes: make(map[uint64]*Node),
		stop:  make(chan struct{}),
	}
}

func (m *NodeManager) AddNode(id uint64) {
	node := NewNode(id, m)
	m.nodes[id] = node
}

func (m *NodeManager) ConnectNodes(i, j, cost uint64) {
	m.nodes[i].AddNeighbor(j, cost)
	m.nodes[j].AddNeighbor(i, cost)
}

func (m *NodeManager) UpdateConnectCost(i, j, cost uint64) {
	m.nodes[i].UpdateNeighborCost(j, cost)
	m.nodes[j].UpdateNeighborCost(i, cost)
}

func (m *NodeManager) Init(table [][]uint64) bool {
	rowsCnt := len(table)
	for i := 0; i < rowsCnt; i++ {
		m.AddNode(uint64(i))
	}

	for i := 0; i < rowsCnt; i++ {
		if len(table[i]) != rowsCnt {
			log.Printf("Invalid table size: rows [%d],row[%d]: %d\n", rowsCnt, i, len(table[i]))
			return false
		}
		for j := 0; j < len(table[i]); j++ {
			if table[i][j] != table[j][i] {
				log.Printf("Invalid tables table[%d][%d] = %d != table[%d][%d] = [%d]\n", i, j, table[i][j], j, i, table[j][i])
				return false
			}
			if j > i && table[i][j] != 0 {
				m.ConnectNodes(uint64(i), uint64(j), table[i][j])
			}
		}
	}

	return true
}

func (m *NodeManager) Start() {
	for _, node := range m.nodes {
		m.wg.Add(1)
		go node.Run(m.nodes, &m.wg, m.stop)
	}
}

func (m *NodeManager) Stop() {
	close(m.stop)
	m.wg.Wait()
}

func (m *NodeManager) ShowTables() {
	for _, node := range m.nodes {
		node.ShowInfo()
	}
}
