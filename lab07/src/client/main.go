package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type TaskType int

const (
	serverAddr = "localhost:8080"
	timeout    = time.Second
	bufferSize = 1024
	timeFormat = "15:04:05"
	packetsCnt = 10

	Ping TaskType = iota
	Heartbeat
)

type Options struct {
	mode    TaskType
	clients int
}

func ParseFlag() (Options, error) {
	var options Options

	modeStr := flag.String("m", "echo", "`echo` for echo server, `heartbeat` for heartbeat tracking")
	clients := flag.Int("c", 2, "number of clients -- only for `heartbeat`")

	flag.Parse()

	switch *modeStr {
	case "echo":
		options.mode = Ping
	case "heartbeat":
		options.mode = Heartbeat
	default:
		return Options{}, errors.New("unknown mode")
	}

	options.clients = *clients

	return options, nil
}

func main() {

	options, err := ParseFlag()
	if err != nil {
		log.Printf("Error parsing flag: %s\n", err)
		os.Exit(1)
	}

	switch options.mode {
	case Ping:
		runPing(true)
	case Heartbeat:
		runHeartbeat(options.clients)
	default:
		log.Printf("Not implemented mod: %d\n", options.mode)
		os.Exit(1)
	}
}

func runPing(isEcho bool) {

	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		log.Printf("Error connecting to server: %s\n", err)
		os.Exit(1)
	}

	log.Printf("Client addr: %s\n", conn.LocalAddr())

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Printf("Error closing connection: %s\n", err)
		}
	}(conn)

	lostPackets := 0
	var rtts []int64
	var msgBegin string
	if isEcho {
		msgBegin = "Ping "
	} else {
		msgBegin = ""
	}

	for i := 1; i <= packetsCnt; i++ {
		msg := fmt.Sprintf("%s%d %s", msgBegin, i, time.Now().Format(timeFormat))

		start := time.Now()
		if _, err := conn.Write([]byte(msg)); err != nil {
			log.Printf("Error writing to server: %s\n", err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			log.Printf("Error setting read deadline: %s\n", err)
			continue
		}

		buf := make([]byte, bufferSize)
		n, err := conn.Read(buf)
		if err != nil {
			lostPackets++
			fmt.Printf("Ping %d: Request timed out\n", i)
		} else {
			rtt := time.Since(start).Microseconds()
			rtts = append(rtts, rtt)
			fmt.Printf("Ping %d: Response time: %dms seconds, Response: [%s]\n", i, rtt, string(buf[:n]))
		}
	}

	showStats(rtts, lostPackets, packetsCnt)
}

func runHeartbeat(clients int) {
	fmt.Println("Heartbeat started")

	wg := sync.WaitGroup{}
	wg.Add(clients)

	for i := 0; i < clients; i++ {
		go func() {
			defer wg.Done()
			runPing(false)
		}()
	}

	wg.Wait()
}

func showStats(rtts []int64, lostPackets, totalPackets int) {
	fmt.Printf("%d packets transmitted, %d received, %.2f%% packet loss\n",
		totalPackets, totalPackets-lostPackets, float64(lostPackets)/float64(totalPackets)*100)

	if len(rtts) == 0 {
		fmt.Printf("No info about RTT\n")
		return
	}
	minRTT := rtts[0]
	maxRTT := rtts[0]
	var sumRTT int64 = 0

	for _, rtt := range rtts {
		sumRTT += rtt
		if rtt < minRTT {
			minRTT = rtt
		}
		if rtt > maxRTT {
			maxRTT = rtt
		}
	}
	avg := sumRTT / int64(len(rtts))

	fmt.Printf("rtt min/avg/max = %d/%d/%d ns\n", minRTT, avg, maxRTT)
}
