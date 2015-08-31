package phantom_tcp

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Config struct {
	Host          string
	Port          uint32
	Net           string
	SendBuffer    uint32
	ReceiveBuffer uint32
	Deadline      *time.Duration
}

func (c *Config) tcpAddr() (*net.TCPAddr, error) {
	return net.ResolveTCPAddr(c.Net, fmt.Sprintf("%s:%d", c.Host, c.Port))
}

type Server struct {
	config  *Config
	handler Handler
	//protocol  Protocol
	chExit    chan bool
	listener  *net.TCPListener
	waitGroup *sync.WaitGroup
}

func NewServer(config *Config, handler Handler) *Server {
	return &Server{
		config:    config,
		handler:   handler,
		chExit:    make(chan bool),
		waitGroup: &sync.WaitGroup{},
	}
}

func (s *Server) Start() {
	s.waitGroup.Add(1)
	defer func() {
		s.listener.Close()
		s.waitGroup.Done()
	}()

	// init listener
	s.listener = net.TCPListener(s.config.Net, s.config.tcpAddr())

	for {
		select {
		case <-s.chExit:
			return
		}

		s.listener.SetDeadline(time.Now().Add(s.config.deadline))

		conn, err := s.listener.AcceptTCP()
		if err != nil {
			continue
		}

		go newConn(conn, s)
	}
}

func (s *Server) Stop() {
	close(s.chExit)
	s.waitGroup.Wait()
}
