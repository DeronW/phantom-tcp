package phantom_tcp

import (
	"net"
	"sync"
	"time"
)

type Config struct {
	sendLimit    uint32
	receiveLimit uint32
}

type Server struct {
	config    *Config
	callback  Callback
	protocol  Protocol
	exitChan  chan int8
	waitGroup *sync.WaitGroup
}

func NewServer(config *Config, callback Callback, protocol Protocol) *Server {
	return &Server{
		config:    config,
		callback:  callback,
		protocol:  protocol,
		exitChan:  make(chan int),
		waitGroup: &sync.WaitGroup{},
	}
}

func (s *Server) Start(listener *net.TCPListener, acceptTime time.Duration) {
	s.waitGroup.Add(1)
	defer func() {
		listener.Close()
		s.waitGroup.Done()
	}()

	for {
		select {
		case <-s.exitChan:
			return
		}

		listener.SetDeadline(time.Now().Add(acceptTime))

		conn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}

		go newConn(conn, s).Do()
	}
}

func (s *Server) Stop() {
	close(s.exitChan)
	s.waitGroup.Wait()
}
