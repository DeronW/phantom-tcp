Phantom TCP
==========

使用GO语言开发的 TCP 服务框架. 与原生TCP服务相比, 封装了消息序列化功能, 
并且针对具体的业务场景做了相应优化

based on https://github.com/gansidui/gotcp

based on https://github.com/felixge/tcpkeepalive

```go
package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "runtime"
    "syscall"
    "time"

    "github.com/gansidui/gotcp"
    "github.com/gansidui/gotcp/examples/telnet"
)

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    // creates a tcp listener
    tcpAddr, err := net.ResolveTCPAddr("tcp4", ":23")
    checkError(err)
    listener, err := net.ListenTCP("tcp", tcpAddr)
    checkError(err)

    // creates a server
    config := &gotcp.Config{
        PacketSendChanLimit:    20,
        PacketReceiveChanLimit: 20,
    }
    srv := gotcp.NewServer(config, &telnet.TelnetCallback{}, &telnet.TelnetProtocol{})

    // starts service
    go srv.Start(listener, time.Second)
    fmt.Println("listening:", listener.Addr())

    // catchs system signal
    chSig := make(chan os.Signal)
    signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
    fmt.Println("Signal: ", <-chSig)

    // stops service
    srv.Stop()
}

func checkError(err error) {
    if err != nil {
        log.Fatal(err)
    }
}
```
