// refer: https://github.com/felixge/tcpkeepalive
// only for Linux
package phantom_tcp

import (
	"net"
	"os"
	"syscall"
	"time"
)

type KeepAliveConfig struct {
	KeepAlive         bool
	KeepAliveIdle     time.Duration
	KeepAliveCount    int
	KeepAliveInterval time.Duration
}

func SetKeepAlive(c *net.TCPConn, cfg *KeepAliveConfig) error {
	if err := c.SetKeepAlive(cfg.KeepAlive); err != nil {
		return err
	}

	file, err := c.File()
	if err != nil {
		return err
	}

	fd := int(file.Fd())

	if cfg.KeepAliveIdle != 0 {
		if err := setIdle(fd, secs(cfg.KeepAliveIdle)); err != nil {
			return err
		}
	}

	if cfg.KeepAliveCount != 0 {
		if err := setCount(fd, cfg.KeepAliveCount); err != nil {
			return err
		}
	}

	if cfg.KeepAliveInterval != 0 {
		if err := setInterval(fd, secs(cfg.KeepAliveInterval)); err != nil {
			return nil
		}
	}

	return nil
}

func secs(t time.Duration) int {
	t += (time.Second - time.Nanosecond)
	return int(t.Seconds())
}

func setIdle(fd int, n int) error {
	return os.NewSyscallError("setsockopt",
		syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPIDLE, n))
}

func setCount(fd int, n int) error {
	return os.NewSyscallError("setsockopt",
		syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPCNT, n))
}

func setInterval(fd int, n int) error {
	return os.NewSyscallError("setsockopt",
		syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL, n))
}
