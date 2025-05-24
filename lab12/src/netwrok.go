package main

import (
	"fmt"
	"sync"
)

type Network struct {
	Routers map[string]*Router
	queue   UnreadMessageQueue
}

func NewNetwork() *Network {
	return &Network{
		Routers: make(map[string]*Router),
		queue:   UnreadMessageQueue{size: 0},
	}
}

type UnreadMessageQueue struct {
	mu   sync.Mutex
	size int
}

func (u *UnreadMessageQueue) Add(x int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.size += x
}

func (u *UnreadMessageQueue) Minus(x int) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.size -= x
}

func (u *UnreadMessageQueue) IsEmpty() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.size == 0
}

func (n *Network) AddRouter(IP string) error {
	if _, ok := n.Routers[IP]; ok {
		return fmt.Errorf("router %s already exists", IP)
	}
	n.Routers[IP] = NewRouter(IP, &n.queue)
	return nil
}

func (n *Network) AddEdge(IP1, IP2 string) error {
	if _, ok := n.Routers[IP1]; !ok {
		return fmt.Errorf("router %s does not exists", IP1)
	}
	if _, ok := n.Routers[IP2]; !ok {
		return fmt.Errorf("router %s does not exists", IP2)
	}
	n.Routers[IP1].AddNeighbor(n.Routers[IP2])
	n.Routers[IP2].AddNeighbor(n.Routers[IP1])
	return nil
}

func (n *Network) Run() {
	wg := sync.WaitGroup{}
	for _, router := range n.Routers {
		wg.Add(1)
		go func() {
			router.Run()
			wg.Done()
		}()
	}
	wg.Wait()
}
