package mqweb

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/iotalking/mqtt-broker/client"
	"github.com/iotalking/mqtt-broker/session"
)

type Server struct {
	clt       *client.Client
	hMap      map[string]Handler
	mtxHMap   sync.Mutex
	closeChan chan byte
	qos       byte
	timeout   time.Duration
}

type Handler func(id string, params []byte)

func NewServer(id string) *Server {
	mgrOnce.Do(func() {
		mgr = session.GetMgr()
	})
	id = fmt.Sprintf("svr:%s", id)
	return &Server{
		clt:       client.NewClient(id, mgr),
		hMap:      make(map[string]Handler),
		closeChan: make(chan byte),
		qos:       0,
		timeout:   30 * time.Second,
	}
}

func (this *Server) SetTimeout(d time.Duration) {
	this.timeout = d
}
func (this *Server) SetQos(qos byte) {
	this.qos = qos
}
func (this *Server) DialGeteWay(gwAddr string) (err error) {
	token, err := this.clt.Connect("mqtt", gwAddr)
	if err != nil {
		return
	}
	if !token.WaitTimeout(this.timeout * time.Second) {
		err = fmt.Errorf("server DialGateWay timeout")
		return
	}
	this.clt.SetOnMessage(this.onmessage)
	return
}

func (this *Server) Listen() {
	select {
	case <-this.closeChan:
	}
	close(this.closeChan)
	this.clt.Disconnect()
}

func (this *Server) Close() {
	this.clt.Disconnect()

	select {
	case this.closeChan <- 1:
	default:
	}
}

func (this *Server) onmessage(id string, data []byte, qos byte) {
	this.mtxHMap.Lock()
	h, ok := this.hMap[id]
	this.mtxHMap.Unlock()
	if ok {
		if len(data) < 4 {
			fmt.Println("len(data) < 4")
			return
		}
		idlen := binary.LittleEndian.Uint32(data)
		if int(idlen) > (len(data) - 4) {
			fmt.Println("int(idlen) > (len(data) - 4)")
			return
		}
		cbid := data[4 : 4+idlen]
		go h(string(cbid), data[4+idlen:])
	}
}
func (this *Server) Serv(url string, handler Handler) (err error) {
	token, err := this.clt.Subcribe(map[string]byte{
		url: 1,
	})
	if err != nil {
		return
	}
	if !token.WaitTimeout(this.timeout * time.Second) {
		err = fmt.Errorf("serv timeout")
		return
	}
	this.mtxHMap.Lock()
	this.hMap[url] = handler
	this.mtxHMap.Unlock()
	return
}
func (this *Server) Callback(id string, result []byte) (err error) {
	token, err := this.clt.Publish(id, result, this.qos, false)
	if err != nil {
		return
	}
	if !token.WaitTimeout(this.timeout * time.Second) {
		err = fmt.Errorf("timeout")
		return
	}
	return
}
