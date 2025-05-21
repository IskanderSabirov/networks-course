package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	port := 8888
	conn, err := net.Dial("tcp6", fmt.Sprintf("[::1]:%d", port))
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println("Close connection error:", err)
		}
	}(conn)

	fmt.Print("Type message: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')

	_, err = conn.Write([]byte(text))
	if err != nil {
		fmt.Println("Sending error:", err)
		return
	}

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Get error:", err)
		return
	}

	fmt.Println("Server response:", string(buffer[:n]))
}
