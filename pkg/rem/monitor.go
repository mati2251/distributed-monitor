package rem

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	// zmq "github.com/pebbe/zmq4"
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

func ReadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	content = []byte(os.ExpandEnv(string(content)))
	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}

type Context struct {
	config Config
}

type Cond struct {
	cond *sync.Cond
}

func NewCond() (*Cond, error) {
	c := &Cond{
		cond: sync.NewCond(&sync.Mutex{}),
	}
	return c, nil
}

func (c *Cond) lock() {
	c.cond.L.Lock()
}

func (c *Cond) unlock() {
	c.cond.L.Unlock()
}

func (c *Cond) Wait() {
	c.cond.Wait()
}

func (c *Cond) SignalAll() {
	c.cond.Broadcast()
}

func (c *Cond) Signal() {
	c.cond.Signal()
}

type Monitor[T any] struct {
	c   *Cond
	val T
}

func NewMonitor[T any](val T) (*Monitor[T], error) {
	m := &Monitor[T]{
		c:   &Cond{cond: sync.NewCond(&sync.Mutex{})},
		val: val,
	}
	return m, nil
}

type SynchronizedFunc[T any] func(m *Cond, val T) error

func (m Monitor[T]) Synchronized(f SynchronizedFunc[T]) error {
	m.c.lock()
	defer m.c.unlock()
	return f(m.c, m.val)
}
