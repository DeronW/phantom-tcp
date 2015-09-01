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
)

type Conn struct {
	Id        uint32
	cfg       *ConnConfig
	conn      *net.TCPConn
	closeOnce sync.Once
	closeFlag int32
	chClose   chan bool
	chSend    chan []byte
	chReceive chan []byte
}

type ConnConfig struct {
	SendBuf    uint32
	ReceiveBuf uint32
	Handler    Handler
	WaitGroup  *sync.WaitGroup
	ExitChan   chan bool
}

func newConn(conn *net.TCPConn, cfg *ConnConfig) {
	c := &Conn{
		cfg:       cfg,
		conn:      conn,
		chClose:   make(chan bool),
		chSend:    make(chan []byte, cfg.SendBuf),
		chReceive: make(chan []byte, cfg.ReceiveBuf),
	}

	if !cfg.Handler.OnConnect(c) {
		return
	}

	go c.loop()
}

func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.closeFlag, 1)
		close(c.chClose)
		c.conn.Close()
		c.cfg.Handler.OnClose(c)
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
	c.cfg.WaitGroup.Add(1)
	defer func() {
		recover()
		c.Close()
		c.cfg.WaitGroup.Done()
	}()

	reader := bufio.NewReader(c.conn)

	for {
		select {
		case <-c.cfg.ExitChan:
			return
		case <-c.chClose:
			return
		case m := <-c.chSend:
			if _, err := c.conn.Write(m); err != nil {
				return
			}
		}

		m, err := reader.ReadBytes('\n')

		if err != nil {
			return
		}

		c.cfg.Handler.OnMessage(c, m)
	}
}
