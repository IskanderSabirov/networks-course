package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TaskType int

const (
	bufferSize    = 2048
	packetLoss    = 5
	defaultPort   = 8080
	LostThreshold = 0.2

	Echo TaskType = iota
	Heartbeat

	timeFormat = "15:04:05"
)

type Params struct {
	Mode      TaskType
	Threshold int
}

func ParseFlags() Params {

	modeStr := flag.String("m", "echo", "`echo` for echo server, `heartbeat` for heartbeat tracking")
	threshold := flag.Int("t", 3, "time in seconds to consider client disconnected (only in heartbeat mode)")

	flag.Parse()

	var mode TaskType
	if *modeStr == "heartbeat" {
		mode = Heartbeat
	} else {
		mode = Echo
	}
	fmt.Printf("mode: %s\n", *modeStr)

	return Params{Mode: mode, Threshold: *threshold}
}

func main() {

	params := ParseFlags()

	addr := net.UDPAddr{IP: net.ParseIP("localhost"), Port: defaultPort}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("failed to bind to %s: %v", addr.String(), err)
	}
	defer func(conn *net.UDPConn) {
		err := conn.Close()
		if err != nil {
			log.Printf("failed to close UDP connection: %v", err)
		}
	}(conn)

	log.Printf("Server running on %s", addr.String())

	switch params.Mode {
	case Heartbeat:
		runHeartbeat(conn, params.Threshold)
	case Echo:
		runEcho(conn)
	default:
		log.Println("Unsupported mode")
		os.Exit(1)
	}
}

func runEcho(conn *net.UDPConn) {
	log.Println("Echo mode started")
	buf := make([]byte, bufferSize)
	packetCnt := 0

	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}

		packetCnt += 1

		log.Printf("[%d] Received %d bytes from %s", packetCnt, n, addr)

		if packetCnt%packetLoss == 0 {
			log.Printf("[%d] Simulated loss for %s", packetCnt, addr)
			continue
		}

		resp := strings.ToUpper(string(buf[:n]))
		if _, err = conn.WriteTo([]byte(resp), addr); err != nil {
			log.Printf("[%d] Write error: %v", packetCnt, err)
		} else {
			log.Printf("[%d] Responded to %s", packetCnt, addr)
		}
	}
}

type Client struct {
	firstPacket    int
	lastPacket     int
	lastPacketTime time.Time
	lostPacketsCnt int
}

type Packet struct {
	idx  int
	time time.Time
}

type Clients struct {
	clients map[string]*Client
	m       sync.Mutex
}

type Update struct {
	addr *net.UDPAddr
	buf  []byte
}

func runHeartbeat(conn *net.UDPConn, threshold int) {
	log.Println("Heartbeat mode started")

	clients := NewClients()
	updates := make(chan Update, 5)
	var exit chan struct{}

	go receiveUpdates(conn, updates, exit)

	thresholdDur := time.Second * time.Duration(threshold)
	removingTimer := time.NewTicker(thresholdDur / time.Duration(3))

	for {
		select {
		case <-removingTimer.C:
			clients.deleteDisconnected(thresholdDur)
		case up := <-updates:
			handleUpdate(conn, up, clients)
		}
	}
}

func receiveUpdates(conn *net.UDPConn, updates chan<- Update, closed <-chan struct{}) {
	buf := make([]byte, bufferSize)
	for {
		// для заврешения горутины
		select {
		case <-closed:
			return
		default:
		}

		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}
		updates <- Update{addr, buf[:n]}
	}
}

func handleUpdate(conn *net.UDPConn, update Update, clients *Clients) {

	packet, err := parsePacket(update.buf)
	if err != nil {
		fmt.Printf("Error parsing packet: %s\n", err.Error())
		return
	}

	if rand.Float32() < LostThreshold {
		log.Printf("Lost packet from [%s] for simulation\n", update.addr)
		return
	}

	clients.handleUpdate(update.addr.String(), packet)

	resp := fmt.Sprintf("%d %s", packet.idx, time.Now().Format(timeFormat))

	if _, err := conn.WriteTo([]byte(resp), update.addr); err != nil {
		log.Printf("Write error: %v", err)
		return
	}
	log.Printf("Sent response %s", resp)
}

func parsePacket(body []byte) (Packet, error) {
	str := string(body)
	parts := strings.Split(str, " ")

	if len(parts) != 2 {
		return Packet{}, errors.New("invalid packet format")
	}

	var packet Packet

	if idx, err := strconv.Atoi(parts[0]); err != nil {
		return Packet{}, errors.New("invalid packet format - idx")
	} else {
		packet.idx = idx
	}

	if packetTime, err := time.Parse(timeFormat, parts[1]); err != nil {
		return Packet{}, errors.New("invalid packet format - time")
	} else {
		packet.time = packetTime
	}

	return packet, nil
}

func NewClients() *Clients {
	return &Clients{
		clients: make(map[string]*Client),
	}
}

func (c *Clients) deleteDisconnected(threshold time.Duration) {
	c.m.Lock()
	defer c.m.Unlock()
	for addr, cl := range c.clients {
		if time.Since(cl.lastPacketTime) > threshold {
			delete(c.clients, addr)
			log.Printf("Deleted connection [%s], now time: %s, last packet time: %s",
				addr, time.Now().Format(timeFormat), cl.lastPacketTime)
		}
	}
}

func (c *Clients) handleUpdate(addr string, packet Packet) {
	c.m.Lock()
	defer c.m.Unlock()

	if cl, find := c.clients[addr]; find {
		newLost := packet.idx - cl.lastPacket - 1
		c.clients[addr].lastPacket = packet.idx
		c.clients[addr].lastPacketTime = packet.time
		c.clients[addr].lostPacketsCnt += newLost
		LogClientInfo(addr, *cl)
	} else {
		c.clients[addr] = &Client{
			firstPacket:    packet.idx,
			lastPacket:     packet.idx,
			lastPacketTime: packet.time,
			lostPacketsCnt: 0,
		}
		LogClientInfo(addr, *c.clients[addr])
	}

	return
}

func LogClientInfo(addr string, client Client) {

	lost := 0.0
	delta := client.lastPacket - client.firstPacket
	if delta > 0 {
		lost = float64(client.lostPacketsCnt) / float64(delta+1) * 100
	}

	log.Printf(
		"%s status: first packet idx: %d, last packet idx: %d,lost %f%%",
		addr, client.firstPacket, client.lastPacket, lost)
}
