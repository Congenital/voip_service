package main

import "net"
import "log"
import "runtime"
import "time"
import "os"
import "strconv"

const HOST = "127.0.0.1"
const PORT = 23000

var c chan bool

func receive(uid int64) {
	ip := net.ParseIP(HOST)
	addr := net.TCPAddr{ip, PORT, ""}

	conn, err := net.DialTCP("tcp4", nil, &addr)
	if err != nil {
		log.Println("connect error")
		c <- false
		return
	}
	seq := 1

	SendMessage(conn, &Message{MSG_AUTH, seq, &Authentication{uid}})
	ReceiveMessage(conn)
	for {
		begin := time.Now().Unix()
		conn.SetDeadline(time.Now().Add(60 * 8 * time.Second))
		msg := ReceiveMessage(conn)
		if msg == nil {
			end := time.Now().Unix()
			if end-begin < 60*7 {
				break
			}
			log.Println("send heartbeat message")
			seq++
			conn.SetDeadline(time.Now().Add(60 * 8 * time.Second))
			SendMessage(conn, &Message{MSG_HEARTBEAT, seq, nil})
			continue
		}
		seq++
		ack := &Message{MSG_ACK, seq, MessageACK(msg.seq)}
		SendMessage(conn, ack)
	}
	conn.Close()
	c <- true
}

func main() {
	runtime.GOMAXPROCS(4)

	log.SetFlags(log.Lshortfile | log.LstdFlags)
	c = make(chan bool, 100)
	var i int64
	var j int64

	if len(os.Args) < 3 {
		log.Println("benchmark_connection first last")
		return
	}

	first, _ := strconv.ParseInt(os.Args[1], 10, 64)
	last, _ := strconv.ParseInt(os.Args[2], 10, 64)

	log.Printf("first:%d last:%d", first, last)
	for i = first; i < last; i += 1000 {
		for j = i; j < i+1000 && j < last; j++ {
			go receive(j)
		}
		time.Sleep(2 * time.Second)
	}

	for i = first; i < last; i++ {
		<-c
	}
}
