package rem

import (
	"fmt"
	"slices"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func NewMonitor[T any](val T, configPath string) (*Monitor[T], error) {
	config, err := ReadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	zctx, err := zmq.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create zmq context: %w", err)
	}

	sub, err := zctx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber socket: %w", err)
	}
	selfHost := ""
	for _, peer := range config.Peers {
		if peer.Id == config.Self {
			selfHost = peer.Host
			continue
		}
		err = sub.Connect(peer.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to connect zmq subscriber socket to peer %s: %w", peer.Host, err)
		}
	}
	err = sub.SetSubscribe(fmt.Sprintf("%d", config.Self))
	if err != nil {
		return nil, fmt.Errorf("failed to set zmq subscription for self: %w", err)
	}
	err = sub.SetSubscribe("0")
	if err != nil {
		return nil, fmt.Errorf("failed to set zmq subscription for broadcast: %w", err)
	}
	err = sub.SetSubscribe("S")
	if err != nil {
		return nil, fmt.Errorf("failed to set zmq subscription for broadcast: %w", err)
	}
	err = sub.SetSubscribe("W")
	if err != nil {
		return nil, fmt.Errorf("failed to set zmq subscription for broadcast: %w", err)
	}
	pub, err := zctx.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf("failed to create zmq publisher socket: %w", err)
	}
	err = pub.Bind(selfHost)
	if err != nil {
		return nil, fmt.Errorf("failed to bind zmq publisher socket: %w", err)
	}
	time.Sleep(100 * time.Millisecond)

	hasToken := false
	if config.Self == config.Token {
		hasToken = true
	}

	rn := make(map[int]int)
	lp := make(map[int]int)
	for _, peer := range config.Peers {
		rn[peer.Id] = 0
		lp[peer.Id] = 0
	}

	mutex := sync.Mutex{}
	cond := sync.NewCond(&mutex)

	m := &Monitor[T]{
		cond:     cond,
		mutex:    &mutex,
		Data:     val,
		rn:       rn,
		sub:      sub,
		pub:      pub,
		zctx:     zctx,
		config:   *config,
		hasToken: hasToken,
		locked:   false,
		wait:     false,
		token: Token{
			Lp: lp,
			Q:  make([]int, 0),
		},
		waiting: make([]int, 0),
	}
	return m, nil
}

func (m *Monitor[T]) Signal() {
	m.sendSignal(false)
}

func (m *Monitor[T]) lock() {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	m.plock()
}

func (m *Monitor[T]) plock() {
	m.rn[m.config.Self]++
	if !m.hasToken {
		m.sendRequest()
		m.cond.Wait()
	}
	m.locked = true
}

func (m *Monitor[T]) unlock() {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	m.punlock()
}

func (m *Monitor[T]) punlock() {
	m.token.Lp[m.config.Self] = m.rn[m.config.Self]
	for id, lp := range m.token.Lp {
		if lp < m.rn[id] && !slices.Contains(m.token.Q, id) && id != m.config.Self {
			m.token.Q = append(m.token.Q, id)
		}
	}
	m.sendToken()
	m.locked = false
}

func (m *Monitor[T]) Wait() {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	m.wait = true
	m.sendWait()
	m.punlock()
	m.cond.Wait()
}

func (m *Monitor[T]) SignalAll() {
	m.sendSignal(true)
}

func (m *Monitor[T]) Synchronized(f SynchronizedFunc[T]) error {
	m.lock()
	defer m.unlock()
	return f(m)
}
