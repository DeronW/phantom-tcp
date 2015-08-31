package phantom_tcp

import (
	"bufio"
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

type connection struct {
	server    *Server
	conn      *net.TCPConn
	closeOnce sync.Once
	closeFlag int32
	chClose   chan bool
	chSend    chan []byte
	chReceive chan []byte
}

func newConn(conn *net.TCPConn, server *Server) {

	c := &connection{
		server:    server,
		conn:      conn,
		chClose:   make(chan bool),
		chSend:    make(chan []byte, sendBuf),
		chReceive: make(chan []byte, receiveBuf),
	}

	if !c.handler.OnConnect(c) {
		return nil
	}

	go c.loop()

	return c
}

func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.closeFlag, 1)
		close(c.chClose)
		c.conn.Close()
		c.handler.OnClose(c)
	})
}

func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closeFlag) == 1
}

func (c *Conn) Write(m []byte, timeout time.Duration) error {
	if c.IsClosed() {
		return ErrConnClosing
	}

	if timeout == 0 {
		select {
		case c.chSend <- m:
			return nil
		default:
			return ErrWriteBlocking
		}
	} else {
		select {
		case c.chSend <- m:
			return nil
		case <-c.chClose:
			return ErrConnClosing
		case <-time.After(timeout):
			return ErrWriteBlocking
		}
	}
}

func (c *Conn) loop() {
	c.server.waitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.server.waitGroup.Done()
	}()

	reader := bufio.NewReader(c)

	for {
		select {
		case <-c.server.chExit:
			return
		case <-c.chClose:
			return
		case m := <-c.chSend:
			if _, err := c.conn.Write(m); err != nil {
				return
			}
		}

		m, err := reader.ReadString('\n')

		if err != nil {
			return
		}

		c.server.handler.OnMessage(m)
	}
}
