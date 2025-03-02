package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	addrFlag := flag.String("h", "localhost", "http service address")
	portFlag := flag.Int("p", 8080, "server port")
	fileFlag := flag.String("f", "./", "file path")

	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(1)
	}

	runClient(*addrFlag, *portFlag, *fileFlag)
}

func runClient(host string, port int, filename string) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatalf("Failed to connect to server: %s", err.Error())
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("Failed to close connection: %s", err.Error())
		}
	}(conn)

	request := fmt.Sprintf("GET /%s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", filename, host)
	_, err = conn.Write([]byte(request))
	if err != nil {
		log.Fatalf("Failed to send request: %s", err.Error())
	}

	response, err := io.ReadAll(conn)
	if err != nil {
		log.Fatalf("Failed to read response: %s", err.Error())
	}

	fmt.Println(string(response))
}
