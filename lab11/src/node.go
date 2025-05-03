package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type RouteInfo struct {
	Next uint64
	Cost uint64
}

type Update struct {
	src        uint64
	RouteInfos map[uint64]uint64 // from dst to cost
}

type Node struct {
	ID        uint64
	Neighbors map[uint64]uint64    // from Node to Cost
	Routes    map[uint64]RouteInfo // from id to len
	ComeInfo  chan Update
	mu        sync.Mutex
	m         *NodeManager
}

func NewNode(id uint64, m *NodeManager) *Node {
	return &Node{
		ID:        id,
		Neighbors: make(map[uint64]uint64),
		Routes:    make(map[uint64]RouteInfo),
		ComeInfo:  make(chan Update, 50),
		m:         m,
	}
}

func (node *Node) AddNeighbor(n uint64, cost uint64) {
	node.mu.Lock()
	defer node.mu.Unlock()

	node.Neighbors[n] = cost

	node.Routes[n] = RouteInfo{n, cost}
}

func (node *Node) UpdateNeighborCost(id, cost uint64) {
	node.mu.Lock()

	prevCost := node.Neighbors[id]
	node.Neighbors[id] = cost

	// обновляем данные
	for dst, routeInfo := range node.Routes {
		if routeInfo.Next == id {
			node.Routes[dst] = RouteInfo{id, routeInfo.Cost - prevCost + cost}
		}
	}

	fmt.Printf("Updated rout cost between %d and %d from %d to %d\n", id, node.ID, prevCost, cost)

	node.mu.Unlock()
	node.SendUpdates(node.m.nodes)
}

func (node *Node) ProcessUpdate(update Update) bool {
	node.mu.Lock()
	defer node.mu.Unlock()
	needSendUpdates := false

	var costToNeighbor uint64
	if cost, exist := node.Neighbors[update.src]; !exist {
		log.Printf("Got update not from registred neighbor: from [%d], to: [%d]\n", node.ID, update.src)
		return false
	} else {
		costToNeighbor = cost
	}

	for dst, costToDst := range update.RouteInfos {
		if dst == node.ID {
			continue
		}

		totalCost := costToNeighbor + costToDst
		curRouteInfo, exist := node.Routes[dst]
		// новый маршрут
		if !exist {
			node.Routes[dst] = RouteInfo{update.src, totalCost}
			needSendUpdates = true
			fmt.Printf("[Node %d] Found new rout to %d with next %d\n", node.ID, dst, update.src)
			continue
		}
		// если выгоднее идти через него теперь
		if curRouteInfo.Cost > totalCost {
			node.Routes[dst] = RouteInfo{update.src, totalCost}
			needSendUpdates = true
			fmt.Printf("[Node %d] Less cost rout to %d with next %d (was: %d, now: %d)]\n", node.ID, dst, update.src, curRouteInfo.Cost, totalCost)
			continue
		}
		// обновляем если шли через него
		if curRouteInfo.Next == update.src && curRouteInfo.Cost != totalCost {
			if neighborCost, exist := node.Neighbors[dst]; exist && neighborCost < totalCost {
				node.Routes[dst] = RouteInfo{dst, neighborCost}
			} else {
				node.Routes[dst] = RouteInfo{update.src, totalCost}
			}
			needSendUpdates = true
			fmt.Printf("[Node %d] Update rout to %d with next %d\n", node.ID, dst, update.src)
			continue
		}
	}

	return needSendUpdates
}

func (node *Node) SendUpdates(m map[uint64]*Node) {
	node.mu.Lock()
	defer node.mu.Unlock()

	update := Update{
		src:        node.ID,
		RouteInfos: make(map[uint64]uint64),
	}

	for dst, info := range node.Routes {
		update.RouteInfos[dst] = info.Cost
	}

	for neighbor := range node.Neighbors {
		m[neighbor].ComeInfo <- update
	}
}

func (node *Node) Run(m map[uint64]*Node, wg *sync.WaitGroup, stop <-chan struct{}) {
	defer wg.Done()

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			node.SendUpdates(m)
		case update := <-node.ComeInfo:
			if node.ProcessUpdate(update) {
				node.SendUpdates(m)
			}
		}
	}
}

func (node *Node) ShowInfo() {
	node.mu.Lock()
	defer node.mu.Unlock()

	fmt.Printf("Node ID: %d\n", node.ID)
	for dst, info := range node.Routes {
		fmt.Printf("Dst:\t%d\tCost:\t%d\tNext node id:\t%d\n", dst, info.Cost, info.Next)
	}
}
