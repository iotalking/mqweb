package mqweb

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iotalking/mqtt-broker/client"
	"github.com/iotalking/mqtt-broker/session"
	"github.com/iotalking/mqtt-broker/utils"
)

var mgr session.SessionMgr
var mgrOnce sync.Once

type Client struct {
	clt       *client.Client
	rchanPool sync.Pool
	cbMap     map[string]Callback
	mtxCbMap  sync.Mutex
	lastId    int64
	qos       byte
	timeout   time.Duration
}
type Callback func([]byte)

type tmData struct {
	rchan chan *struct{}
	timer *time.Timer
}

func NewClient() (c *Client) {
	mgrOnce.Do(func() {
		mgr = session.GetMgr()
	})
	c = &Client{
		clt:     client.NewClient(fmt.Sprintf("clt%s", utils.NewId()), mgr),
		cbMap:   make(map[string]Callback),
		qos:     0,
		timeout: 30,
	}
	c.rchanPool = sync.Pool{
		New: func() interface{} {
			return &tmData{
				rchan: make(chan *struct{}),
				timer: time.NewTimer(c.timeout * time.Second),
			}
		},
	}
	return
}
func (this *Client) NewCbId() (id int64) {
	id = atomic.AddInt64(&this.lastId, 1)
	return
}
func (this *Client) SetTimeout(d time.Duration) {
	this.timeout = d
}
func (this *Client) SetQos(qos byte) {
	this.qos = qos
}

//连接服务器
func (this *Client) DialGateWay(addr string) (err error) {
	_, err = this.clt.Connect("mqtt", addr)
	if err != nil {
		return
	}
	var token session.Token
	token, err = this.clt.Subcribe(map[string]byte{
		fmt.Sprintf("$client/%s/cb/+", this.clt.GetID()): this.qos,
	})
	if err != nil {
		return
	}
	if !token.WaitTimeout(this.timeout * time.Second) {
		err = fmt.Errorf("client DialGateWay timeout")
	}
	this.clt.SetOnMessage(this.onmessage)
	return
}
func (this *Client) Close() (err error) {
	err = this.clt.Disconnect()
	return
}
func (this *Client) onmessage(id string, data []byte, qos byte) {
	var cb Callback
	var ok bool
	this.mtxCbMap.Lock()
	if cb, ok = this.cbMap[id]; ok {
		delete(this.cbMap, id)
	}
	this.mtxCbMap.Unlock()
	if ok {
		go cb(data)
	}
}

func (this *Client) setCalback(id string, cb Callback) (err error) {
	this.mtxCbMap.Lock()
	this.cbMap[id] = cb
	this.mtxCbMap.Unlock()
	return
}
func (this *Client) delCalback(id string) (err error) {
	this.mtxCbMap.Lock()
	delete(this.cbMap, id)
	this.mtxCbMap.Unlock()
	return
}

//回调topic:$client/<clientId>/<callid>
func (this *Client) Call(url string, params []byte) (result []byte, err error) {
	id := fmt.Sprintf("$client/%s/cb/%d", this.clt.GetID(), this.NewCbId())

	td := this.rchanPool.Get().(*tmData)
	td.timer.Reset(this.timeout * time.Second)
	err = this.setCalback(id, func(data []byte) {
		result = data
		td.rchan <- new(struct{})
	})

	idlen := len(id)
	var data = make([]byte, 4+idlen+len(params))
	//lenght of id
	binary.LittleEndian.PutUint32(data, uint32(len(id)))
	//id
	copy(data[4:], []byte(id))
	//params
	copy(data[4+idlen:], params)

	_, err = this.clt.Publish(url, data, this.qos, false)
	if err != nil {
		this.delCalback(id)
		this.rchanPool.Put(td)
		return
	}
	select {
	case <-td.timer.C:
		this.delCalback(id)
		err = fmt.Errorf("call timeout")
	case <-td.rchan:
	}
	this.rchanPool.Put(td)
	return
}
