package rem

import (
	"sync"

	zmq "github.com/pebbe/zmq4"
)

const (
	BRODCAST = "0"
)

type PeerConfig struct {
	Id   int    `json:"id"`
	Host string `json:"host"`
}

type Config struct {
	Self  int          `json:"self"`
	Token int          `json:"token"`
	Peers []PeerConfig `json:"peers"`
}

type Token struct {
	Lp map[int]int
	Q  []int
}

type Monitor[T any] struct {
	Data      T
	cond      *sync.Cond
	mutex     *sync.Mutex
	rn        map[int]int
	timestamp int
	hasToken  bool
	sub       *zmq.Socket
	pub       *zmq.Socket
	zctx      *zmq.Context
	config    Config
	token     Token
	locked    bool
}

type TokenRequest[T any] struct {
	Token Token
	Data  T
}

type Request struct {
	Id        int
	Timestamp int
}

type SynchronizedFunc[T any] func(m *Monitor[T]) error
