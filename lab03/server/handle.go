package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const (
	BufCap      = 2048
	ResourceDir = "examples/"
)

var (
	NotFound      = []byte("HTTP/1.1 404 Not Found" + separator() + separator())
	Ok            = []byte("HTTP/1.1 200 OK" + separator())
	ContentType   = []byte("Content-Type: text/plain; charset=UTF-8" + separator())
	ContentLength = []byte("Content-Length: ")
)

func handleConn(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}(conn)

	buf := make([]byte, BufCap)
	n, err := conn.Read(buf)

	if err != nil && err != io.EOF {
		log.Printf("Error reading connection - %s", err.Error())
		return
	}
	buf = buf[:n]
	//fmt.Println("--------------------------------------------")
	//fmt.Println(string(buf))
	//fmt.Println("--------------------------------------------")
	req := strings.Split(string(buf), " ")
	if len(req) < 2 {
		log.Printf("Error parsing request - %v", req)
		if _, err := conn.Write(NotFound); err != nil {
			log.Printf("Error writing response - %s", err.Error())
			return
		}
		return
	}

	fileName := strings.Trim(req[1], "/")
	fileLength, err := os.ReadFile(ResourceDir + fileName)
	if err != nil {
		log.Printf("Error reading file - %s\n", err.Error())
		if _, err := conn.Write(NotFound); err != nil {
			log.Printf("Error writing response - %s", err.Error())
			return
		}
		return
	}

	n, err = conn.Write([]byte(createOkResponse(fileLength)))
	if err != nil {
		log.Printf("Error writing response: %s\n", err.Error())
		return
	}
	log.Printf("Written response: %d bytes", n)
}

func handlePull(runners chan struct{}, reqs chan net.Conn) {
	for req := range reqs {
		runners <- struct{}{}
		localReq := req
		go func(retChan chan struct{}) {
			handleConn(localReq)
			<-retChan
		}(runners)
	}
}

func separator() string {
	var osSep = fmt.Sprintf("%v", os.PathSeparator)
	var lb = "\n"
	if osSep != "/" {
		lb = "\r\n"
	}
	return lb
}

func createOkResponse(fileData []byte) string {
	return fmt.Sprintf(
		"%s%s%d%s%s%s%s%s",
		Ok,
		ContentLength,
		len(fileData),
		separator(),
		ContentType,
		separator(),
		separator(),
		fileData,
	)
}
