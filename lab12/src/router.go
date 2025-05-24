package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"text/tabwriter"
)

const (
	MaxRoutersMessageQueue = 1024
)

var (
	outputMutex = &sync.Mutex{}
)

type PathToRoute struct {
	NextHop string
	Metric  int
}

type RouterTable map[string]PathToRoute

type RouterMessage struct {
	From  string
	Table RouterTable
}

type Router struct {
	IP              string
	LocalTable      RouterTable
	MessageChannel  chan RouterMessage
	Neighbors       map[string]*Router
	NeighborsTables map[string]RouterTable
	GlobalQueue     *UnreadMessageQueue
}

func NewRouter(IP string, queue *UnreadMessageQueue) *Router {
	return &Router{
		IP:              IP,
		LocalTable:      make(RouterTable),
		MessageChannel:  make(chan RouterMessage, MaxRoutersMessageQueue),
		Neighbors:       make(map[string]*Router),
		NeighborsTables: make(map[string]RouterTable),
		GlobalQueue:     queue,
	}
}

func (router *Router) AddNeighbor(neighbor *Router) {
	router.Neighbors[neighbor.IP] = neighbor
}

func (router *Router) printStatistics(iteration int, isFinal bool) {
	outputMutex.Lock()
	defer outputMutex.Unlock()

	if isFinal {
		fmt.Printf("\nFinal simulation step #%d of router[%s]\n", iteration, router.IP)
	} else {
		fmt.Printf("\nSimulation step #%d of router[%s]\n", iteration, router.IP)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, ' ', 0)
	_, _ = fmt.Fprint(w, "[Source_IP]\t[Destination_IP]\t[Next_Hop]\t[Metric]\n")
	for dst, routeInfo := range router.LocalTable {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", router.IP, dst, routeInfo.NextHop, routeInfo.Metric)
	}
	err := w.Flush()
	if err != nil {
		log.Print(err)
	}
}

func (router *Router) recalculateRoutes() {
	updatedRoutes := make(RouterTable)
	updatedRoutes[router.IP] = PathToRoute{router.IP, 0}

	for _, neighbor := range router.Neighbors {
		for dst, routeInfo := range router.NeighborsTables[neighbor.IP] {
			if _, ok := updatedRoutes[dst]; !ok || updatedRoutes[dst].Metric > routeInfo.Metric+1 {
				updatedRoutes[dst] = PathToRoute{neighbor.IP, routeInfo.Metric + 1}
			}
		}
	}

	if !checkForUpdated(updatedRoutes, router.LocalTable) {
		router.LocalTable = updatedRoutes
		router.sendUpdates()
	}
}

func (router *Router) sendUpdates() {
	router.GlobalQueue.Add(len(router.Neighbors))
	for _, neighbor := range router.Neighbors {
		neighbor.MessageChannel <- RouterMessage{router.IP, router.LocalTable}
	}
}

func checkForUpdated(r1, r2 RouterTable) bool {
	if len(r1) != len(r2) {
		return false
	}

	for key, value := range r1 {
		if value2, ok := r2[key]; !ok || value2 != value {
			return false
		}
	}

	return true
}

func (router *Router) handleMessage(msg RouterMessage) {
	router.NeighborsTables[msg.From] = msg.Table
	router.recalculateRoutes()
	router.GlobalQueue.Minus(1)
}

func (router *Router) Run() {
	router.recalculateRoutes()
	step := 0
	for {
		if router.GlobalQueue.IsEmpty() {
			router.printStatistics(step, true)
			return
		}
		select {
		case message := <-router.MessageChannel:
			router.handleMessage(message)
			step++
			router.printStatistics(step, false)
		default:
		}
	}
}
