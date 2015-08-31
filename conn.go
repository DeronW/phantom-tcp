package phantom_tcp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrConnClosing   = errors.New("conn close")
	ErrWriteBlocking = errors.New("write blocking")
	ErrReadBlocking  = errors.New("read blocking")
)

type Conn struct {
	server            *Server
	conn              *net.TCPConn
	extraData         interface{}
	closeOnce         sync.Once
	closeFlag         int32
	closeChan         chan bool
	packetSendChan    chan Packet
	packetReceiveChan chan Packet
}

type Callback interface {
	OnConnect(*Conn) bool
	OnMessage(*Conn, Packet) bool
	OnClose(*Conn)
}

func newConn(conn *net.TCPConn, server *Server) *Conn {
	return &Conn{
		server:            server,
		conn:              conn,
		closeChan:         make(chan bool),
		packetSendChan:    make(chan Packet, server.config.PacketSendChanLimit),
		packetReceiveChan: make(chan Packet, server.config.PacketReceiveChanLimit),
	}
}

func (c *Conn) GetExtraData() interface{} {
	return c.extraData
}

func (c *Conn) PutExtraData(data interface{}) {
	return c.conn
}

func (c *Conn) GetRawConn() *net.TCPConn {
	return c.conn
}

func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.closeFlag, 1)
		close(c.closeChan)
		c.conn.Close()
		c.server.callback.OnClose(c)
	})
}

func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closeFlag) == 1
}

func (c *Conn) AsyncReadPacket(timeout time.Duration) (Packet, error) {
	if c.IsClosed() {
		return nil, ErrConnClosing
	}

	if timeout == 0 {
		select {
		case p := <-c.packetReceiveChan:
			return p, nil
		default:
			return nil, ErrReadBlocking
		}
	} else {
		select {
		case p := <-c.packetReceiveChan:
			return p, nil
		case <-c.closeChan:
			return nil, ErrConnClosing
		case <-time.After(timeout):
			return nil, ErrReadBlocking
		}
	}
}

func (c *Conn) AsyncWritePacket(p Packet, timeout time.Duration) error {
	if c.IsClose() {
		return ErrConnClosing
	}

	if timeout == 0 {
		select {
		case c.packetSendChan <- p:
			return nil
		default:
			return ErrWriteBlocking
		}
	} else {
		select {
		case c.packetSendChan <- p:
			return nil
		case <-c.closeChan:
			return ErrConnClosing
		case <-time.After(timeout):
			return ErrWriteBlocking
		}
	}
}

func (c *Conn) Do() {
	if !c.server.callback.OnConnect(c) {
		return
	}

	go c.handleLoop()
	go c.readLoop()
	go c.writeLoop()
}

func (c *Conn) readLoop() {
	c.server.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.server.waitGroup.Done()
	}()

	for {
		select {
		case <-c.server.exitChan:
			return
		case <-c.closeChan:
			return
		}

		p, err := c.server.protocol.ReadPacket(c.conn)

		if err != nil {
			return
		}

		c.packetReceiveChan <- p
	}
}

func (c *Conn) writeLoop() {
	c.server.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.server.waitGroup.Done()
	}()

	for {
		select {
		case <-c.server.exitChan:
			return
		case <-c.closeChan:
			return
		case p := <-c.packetSendChan:
			if _, err := c.conn.Write(p.Serialize()); err != nil {
				return
			}
		}
	}
}

func (c *Conn) handleLoop() {
	c.server.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.server.waitGroup.Done()
	}()

	for {
		select {
		case <-c.server.exitChan:
			return
		case <-c.closeChan:
			return
		case p := <-c.packetReceiveChan:
			if !c.server.callback.OnMessage(c, p) {
				return
			}
		}
	}
}
