package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/boisjacques/quic-conn"
	"github.com/tylerwince/godbg"
	"io"
	"math/big"
	"net"
	"os"
	"time"
)

const BUFFERSIZE = 512

func main() {
	// utils.SetLogLevel(utils.LogLevelDebug)

	startServer := flag.Bool("s", false, "server")
	startClient := flag.Bool("c", false, "client")
	var addr string
	flag.StringVar(&addr, "addr", "", "address:port")
	flag.Parse()

	if *startServer {
		// start the server
		go func() {
			tlsConfig, err := generateTLSConfig()
			if err != nil {
				panic(err)
			}

			ln, err := quicconn.Listen("udp", addr, tlsConfig)
			if err != nil {
				panic(err)
			}

			fmt.Println("Waiting for incoming connection")
			conn, err := ln.Accept()
			if err != nil {
				panic(err)
			}
			fmt.Println("Established connection")

			go sendFileToClient(conn)
		}()
	}

	if *startClient {
		// run the client
		go func() {
			 tlsConfig := &tls.Config{InsecureSkipVerify: true}
			conn, err := quicconn.Dial(addr, tlsConfig)
			if err != nil {
				panic(err)
			}
			defer conn.Close()
			fmt.Println("Connected to server, start receiving the file name and file size")
			var fileSize int64
			var fileNameLen int32

			err = binary.Read(conn, binary.BigEndian, &fileSize)
			if err != nil {
				panic(err)
			}
			godbg.Dbg(fileSize)
			err = binary.Read(conn, binary.BigEndian, &fileNameLen)
			if err != nil {
				panic(err)
			}
			godbg.Dbg(fileNameLen)

			err = binary.Write(conn, binary.BigEndian, 1)
			if err != nil {
				panic(err)
			}

			fileNameBuffer := make([]byte, fileNameLen)
			_,err = io.ReadFull(conn, fileNameBuffer)

			fileName := string(fileNameBuffer[:fileNameLen])
			godbg.Dbg(fileName)
			newFile, err := os.Create("recvd_" + fileName)

			if err != nil {
				panic(err)
			}
			defer newFile.Close()
			var receivedBytes int64

			for {
				if (fileSize - receivedBytes) < BUFFERSIZE {
					io.CopyN(newFile, conn, (fileSize - receivedBytes))
					conn.Read(make([]byte, (receivedBytes+BUFFERSIZE)-fileSize))
					break
				}
				io.CopyN(newFile, conn, BUFFERSIZE)
				receivedBytes += BUFFERSIZE
			}
			fmt.Println("Received file completely!")
		}()
	}

	time.Sleep(time.Hour)
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM := pem.EncodeToMemory(&b)

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}, nil
}

func sendFileToClient(connection net.Conn) {
	fmt.Println("Connection Established!")
	defer connection.Close()
	file, err := os.Open("100MB.zip")
	if err != nil {
		fmt.Println(err)
		return
	}
	fileInfo, err := file.Stat()
	godbg.Dbg(fileInfo.Size())
	godbg.Dbg(fileInfo.Name())
	if err != nil {
		fmt.Println(err)
		return
	}
	var fileSize int64
	var fileNameLen int32
	fileSize = fileInfo.Size()
	fileName := fileInfo.Name()
	fileNameLen = int32(len(fileName))
	godbg.Dbg(fileNameLen)
	time.Sleep(10 * time.Millisecond)

	err = binary.Write(connection, binary.BigEndian, fileSize)
	if err != nil {
		panic(err)
	}
	godbg.Dbg("sent filesize")
	err = binary.Write(connection, binary.BigEndian, fileNameLen)
	if err != nil {
		panic(err)
	}
	godbg.Dbg("sent file name length")
	_,err = io.WriteString(connection, fileName)
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
	fmt.Println("File has been sent, closing connection!")
	return
}