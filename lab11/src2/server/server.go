package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	port := 8888
	listener, err := net.Listen("tcp6", fmt.Sprintf("[::1]:%d", port))
	if err != nil {
		fmt.Println("Server start error:", err)
		return
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println("Error in closing listener:", err)
		}
	}(listener)
	fmt.Println("Server listening on:", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Error in closing connection:", err)
		}
	}(conn)
	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Reading error:", err)
		return
	}

	received := string(buffer[:n])
	fmt.Printf("Got: %s\n", received)

	upper := strings.ToUpper(received)
	_, err = conn.Write([]byte(upper))
	if err != nil {
		fmt.Println("Sending error:", err)
		return
	}
}
