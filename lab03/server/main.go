package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

type TaskType int

const (
	TaskA TaskType = iota
	TaskB
	TaskD
	host = "localhost"
)

var limit int = 1

func main() {
	modeFlag := flag.String("t", "A", "task")
	portFlag := flag.Int("p", 8080, "server port")
	boundFlag := flag.Int("l", 1, "concurrency level")

	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}

	taskType := TaskA

	if *modeFlag == "B" {
		taskType = TaskB
	} else if *modeFlag == "D" {
		if *boundFlag <= 1 {
			taskType = TaskB
		} else {
			taskType = TaskD
			limit = *boundFlag
		}
	}

	addr := host + ":" + fmt.Sprintf("%d", *portFlag)
	log.Printf("Serving on %s\n", addr)
	log.Printf("Running mode %s\n", *modeFlag)

	run(addr, taskType)
}

func run(addr string, task TaskType) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Error listening %s - %s", addr, err.Error())
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Printf("Error closing listener - %s", err.Error())
		}
	}(listener)

	var runners chan struct{}
	var req chan net.Conn

	if task == TaskD {
		runners = make(chan struct{}, limit)
		req = make(chan net.Conn)
		go handlePull(runners, req)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting - %s", err.Error())
		}
		log.Printf("Accepted connection %v\n", conn.LocalAddr())

		switch task {
		case TaskA:
			handleConn(conn)
		case TaskB:
			go handleConn(conn)
		case TaskD:
			req <- conn
		default:
			log.Println("Unresolved type of task")
			return
		}
	}
}
