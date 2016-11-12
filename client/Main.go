package main

import "net"

func main() {

	conn, err := net.Dial("tcp", "127.0.0.1:8081")
	if err != nil {
		panic(err)
	}

	conn.Write([]byte("/C/badboy64"))

	l, _ := net.Listen("tcp", "127.0.0.1:8081")
	defer l.Close()
	for {
		conn, err = l.Accept()
		if err != nil {
			panic(err)
		}
	}

}
