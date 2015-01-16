package main

import "time"
import "net"
import log "github.com/golang/glog"

type Peer struct {
	host  string
	port  int
	wt    chan *Message
	conn  *net.TCPConn
	alive bool
}

func (peer *Peer) PeerID() int64 {
	ip := net.ParseIP(peer.host)
	i := int64(ip[0])<<24 | int64(ip[1])<<16 | int64(ip[2])<<8 | int64(ip[3])
	return i<<32 | int64(peer.port)
}

func (peer *Peer) Connected() bool {
	return peer.conn != nil
}

func (peer *Peer) Read() {
	for {
		msg := ReceiveMessage(peer.conn)
		if msg == nil {
			peer.wt <- nil
			break
		}
		log.Info("msg:", msg.cmd)
		if msg.cmd == MSG_IM {
			peer.HandleIMMessage(msg.body.(*IMMessage))
		} else if msg.cmd == MSG_GROUP_IM {
			peer.HandleGroupIMMessage(msg.body.(*IMMessage))
		} else if msg.cmd == MSG_PEER_ACK {
			peer.HandlePeerACK(msg.body.(*MessagePeerACK))
		} else if msg.cmd == MSG_INPUTING {
			peer.HandleInputing(msg.body.(*MessageInputing))
		}
	}
}

func (peer *Peer) HandleInputing(msg *MessageInputing) {
	other := route.FindClient(msg.receiver)
	if other != nil {
		other.wt <- &Message{cmd: MSG_INPUTING, body: msg}
	}
}

func (peer *Peer) HandlePeerACK(msg *MessagePeerACK) {
	other := route.FindClient(msg.receiver)
	if other != nil {
		other.wt <- &Message{cmd: MSG_PEER_ACK, body: msg}
	} else {
		log.Info("can't find client:", msg.receiver)
		storage.SaveOfflineMessage(msg.receiver, &Message{cmd: MSG_PEER_ACK, body: msg})
	}
}

func (peer *Peer) HandleIMMessage(msg *IMMessage) {
	other := route.FindClient(msg.receiver)
	if other != nil {
		other.wt <- &Message{cmd: MSG_IM, body: msg}
	} else {
		log.Info("can't find client:", msg.receiver)
		storage.SaveOfflineMessage(msg.receiver, &Message{cmd: MSG_IM, body: msg})
	}
}

func (peer *Peer) HandleGroupIMMessage(msg *IMMessage) {
	group := group_manager.FindGroup(msg.receiver)
	if group == nil {
		log.Info("can't find group:", msg.receiver)
		return
	}
	for member := range group.Members() {
		other := route.FindClient(member)
		if other != nil {
			other.wt <- &Message{cmd: MSG_GROUP_IM, body: msg}
		} else {
			storage.SaveOfflineMessage(member, &Message{cmd: MSG_GROUP_IM, body: msg})
		}
	}
}

func (peer *Peer) Write() {
	ticker := time.NewTicker(10 * time.Second)
	seq := 0
	for {
		select {
		case msg := <-peer.wt:
			if msg == nil {
				log.Info("socket closed")
				peer.conn = nil
				break
			}
			seq++
			msg.seq = seq
			log.Info("peer msg:", msg.cmd)
			SendMessage(peer.conn, msg)
		case <-ticker.C:
			log.Info("peer send heartbeat")
			seq++
			m := &Message{cmd: MSG_HEARTBEAT, seq: seq}
			SendMessage(peer.conn, m)
		}
		if peer.conn == nil {
			break
		}
	}
}

func (peer *Peer) Start() {
	peer.alive = true
	go peer.Connect()
}

func (peer *Peer) Stop() {
	peer.alive = false
}

func (peer *Peer) AddAllClient() {
	uids := route.GetClientUids()
	for uid, ts := range uids {
		ac := &MessageAddClient{uid, ts}
		msg := &Message{cmd: MSG_ADD_CLIENT, body: ac}
		peer.wt <- msg
	}
}

func (peer *Peer) Connect() {
	ip := net.ParseIP(peer.host)
	addr := net.TCPAddr{ip, peer.port, ""}
	for {
		if peer.conn == nil && peer.alive {
			conn, err := net.DialTCP("tcp4", nil, &addr)
			if err != nil {
				log.Info("connect error:", ip, " ", peer.port)
			} else {
				log.Infof("peer:%s port:%d connected", ip, peer.port)
				peer.conn = conn
				go peer.Read()
				go peer.Write()
				peer.AddAllClient()
			}
		}
		timer := time.NewTimer(20 * time.Second)
		<-timer.C
	}
}
