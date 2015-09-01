package phantom_tcp

// interface of methods that are used as callbacks on a connection
type Handler interface {

	// OnConnect is called when the connection was accepted,
	// If the return value of false is closed
	OnConnect(*Conn) bool

	// OnMessage is called when the connection receives a packet,
	// If the return value of false is closed
	OnMessage(*Conn, []byte) bool

	// OnClose is called when the connection closed
	OnClose(*Conn)
}
