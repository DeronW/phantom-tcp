package phantom_tcp

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type ServerConfig struct {
	Host       string
	Port       uint32
	Net        string
	SendBuf    uint32
	ReceiveBuf uint32

	Deadline          time.Duration
	KeepAlive         bool
	KeepAliveIdle     time.Duration
	KeepAliveCount    int
	KeepAliveInterval time.Duration

	Separtor byte
}

func (c *ServerConfig) tcpAddr() (*net.TCPAddr, error) {
	return net.ResolveTCPAddr(c.Net, fmt.Sprintf("%s:%d", c.Host, c.Port))
}

type Server struct {
	srvCfg  *ServerConfig
	connCfg *ConnConfig
	tcpCfg  *KeepAliveConfig
	//handler Handler
	//protocol  Protocol
	ExitChan  chan bool
	listener  *net.TCPListener
	waitGroup *sync.WaitGroup
}

func NewServer(cfg *ServerConfig, handler Handler) *Server {

	ec := make(chan bool)
	wg := &sync.WaitGroup{}

	connCfg := &ConnConfig{
		SendBuf:    cfg.SendBuf,
		ReceiveBuf: cfg.ReceiveBuf,
		Handler:    handler,
		WaitGroup:  wg,
		ExitChan:   ec,
		Separtor:   cfg.Separtor,
	}

	tcpCfg := &KeepAliveConfig{
		KeepAlive:         cfg.KeepAlive,
		KeepAliveIdle:     cfg.KeepAliveIdle,
		KeepAliveCount:    cfg.KeepAliveCount,
		KeepAliveInterval: cfg.KeepAliveInterval,
	}

	return &Server{
		srvCfg:    cfg,
		tcpCfg:    tcpCfg,
		connCfg:   connCfg,
		ExitChan:  ec,
		waitGroup: wg,
	}
}

func (s *Server) Start() error {
	s.waitGroup.Add(1)
	defer func() {
		s.listener.Close()
		s.waitGroup.Done()
	}()

	// init listener
	addr, err := s.srvCfg.tcpAddr()
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(s.srvCfg.Net, addr)
	if err != nil {
		return err
	}
	s.listener = listener

	for {
		select {
		case <-s.connCfg.ExitChan:
			return nil
		default:
		}

		if s.srvCfg.Deadline != 0 {
			s.listener.SetDeadline(time.Now().Add(s.srvCfg.Deadline))
		}
		conn, err := s.listener.AcceptTCP()

		if err != nil {
			return err // when debug or development, use this.
			//continue
		}

		SetKeepAlive(conn, s.tcpCfg)
		go newConn(conn, s.connCfg)
	}
}

func (s *Server) Stop() {
	close(s.ExitChan)
	s.waitGroup.Wait()
}
