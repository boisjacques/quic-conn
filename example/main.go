package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/tylerwince/godbg"
	"io"
	"net"
	"os"
	"time"

	// "github.com/boisjacques/quic-conn"
)

const BUFFERSIZE = 1024

func main() {
	// utils.SetLogLevel(utils.LogLevelDebug)
	var addr string
	var file string

	startServer := flag.Bool("s", false, "server")
	startClient := flag.Bool("c", false, "client")
	flag.StringVar(&file, "file", "5MB.zip", "filename")
	flag.StringVar(&addr, "addr", "", "address:port")
	flag.Parse()

	finishChan := make(chan bool)

	if *startServer {
		// start the server
		go func() {
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				panic(err)
			}

			fmt.Println("Waiting for incoming connection")
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}
			fmt.Println("Established connection")

			go sendFileToClient(conn, file, finishChan)
		}()
	}

	if *startClient {
		// run the client
		go func() {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				panic(err)
			}
			fmt.Println("Connected to server, start receiving the file name and file size")
			var fileSize int64
			var fileNameLen int32

			err = binary.Read(conn, binary.BigEndian, &fileSize)
			if err != nil {
				panic(err)
			}
			err = binary.Read(conn, binary.BigEndian, &fileNameLen)
			if err != nil {
				panic(err)
			}

			var handshake int32
			handshake = 1
			err = binary.Write(conn, binary.BigEndian, handshake)
			if err != nil {
				panic(err)
			}

			fileNameBuffer := make([]byte, fileNameLen)
			_, err = io.ReadFull(conn, fileNameBuffer)

			fileName := string(fileNameBuffer[:fileNameLen])
			newFile, err := os.Create("recvd_" + fileName)

			if err != nil {
				panic(err)
			}
			defer newFile.Close()
			var receivedBytes int64

			for {
				if (fileSize - receivedBytes) < BUFFERSIZE {
					_, err = io.CopyN(newFile, conn, (fileSize - receivedBytes))
					if err != nil {
						godbg.Dbg(err)
					}
					_, err = conn.Read(make([]byte, (receivedBytes+BUFFERSIZE)-fileSize))
					if err != nil {
						godbg.Dbg(err)
					}
					break
				}
				_, err = io.CopyN(newFile, conn, BUFFERSIZE)
				if err != nil {
					godbg.Dbg(err)
				}
				receivedBytes += BUFFERSIZE
			}
			if err == nil {
				fmt.Println("Received file completely!")
			} else {
				godbg.Dbg(err)
			}
			err = conn.Close()
			if err != nil {
				godbg.Dbg(err)
			}
			finishChan <- true
		}()
	}

	<-finishChan
}

func sendFileToClient(connection net.Conn, f string, finishChan chan bool) {
	fmt.Println("Connection Established!")
	defer connection.Close()
	file, err := os.Open(f)
	if err != nil {
		fmt.Println(err)
		return
	}
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}
	var fileSize int64
	var fileNameLen int32
	fileSize = fileInfo.Size()
	fileName := fileInfo.Name()
	fileNameLen = int32(len(fileName))
	time.Sleep(10 * time.Millisecond)

	err = binary.Write(connection, binary.BigEndian, fileSize)
	if err != nil {
		panic(err)
	}
	err = binary.Write(connection, binary.BigEndian, fileNameLen)
	if err != nil {
		panic(err)
	}
	_, err = io.WriteString(connection, fileName)
	if err != nil {
		panic(err)
	}
	var handshake int32
	err = binary.Read(connection, binary.BigEndian, &handshake)
	if err != nil {
		panic(err)
	}
	if handshake != 1 {
		panic("handshake failed")
	}

	time.Sleep(10 * time.Millisecond)
	sendBuffer := make([]byte, BUFFERSIZE)
	fmt.Println("Start sending file!")
	for {
		_, err = file.Read(sendBuffer)
		if err == io.EOF {
			break
		}
		connection.Write(sendBuffer)
	}
	fmt.Println("File has been sent, waiting 1 second!")
	time.Sleep(1 * time.Second)
	fmt.Println("Closing connection...")
	finishChan <- true
}
