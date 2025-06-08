package rem

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	zmq "github.com/pebbe/zmq4"
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

type Monitor struct {
	cond         Conditinal
	tokenOwner   bool
	lockAcquired bool
	requester    bool
	mutex        sync.Mutex
	sub         *zmq.Socket
	Pub         *zmq.Socket
	zctx        *zmq.Context
	config      *Config
	requesteres []string
	message     chan string
}

func NewMonitor(config Config) (*Monitor, error) {
	s := &Monitor{}
	s.config = &config
	var err error
	s.zctx, err = zmq.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create zmq context: %w", err)
	}

	s.sub, err = s.zctx.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fmt.Errorf("failed to create zmq socket: %w", err)
	}
	selfHost := ""
	for _, peer := range config.Peers {
		if peer.Id == config.Self {
			selfHost = peer.Host
			continue
		}
		err = s.sub.Connect(peer.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to connect zmq subscriber socket to peer %s: %w", peer.Host, err)
		}
	}
	err = s.sub.SetSubscribe(fmt.Sprintf("%d", config.Self))
	if err != nil {
		return nil, fmt.Errorf("failed to set zmq subscription for self: %w", err)
	}
	if config.Self == config.Token {
		err := s.sub.SetSubscribe("token")
		if err != nil {
			return nil, fmt.Errorf("failed to set zmq subscription for token: %w", err)
		}
	}

	s.Pub, err = s.zctx.NewSocket(zmq.PUB)
	if err != nil {
		return nil, fmt.Errorf("failed to create zmq publisher socket: %w", err)
	}
	err = s.Pub.Bind(selfHost)
	if err != nil {
		return nil, fmt.Errorf("failed to bind zmq publisher socket: %w", err)
	}
	return s, nil
}

func (s *Monitor) Close() {
	s.Pub.Close()
	s.sub.Close()
}

func (s *Monitor) Loop() {
	for {
		msg, err := s.sub.RecvMessage(0)
		if err != nil {
			fmt.Printf("failed to receive zmq message: %s\n", err.Error())
			continue
		}
		if len(msg) == 0 {
			continue
		}
		switch msg[0] {
		case fmt.Sprintf("%d", s.config.Self):
			fmt.Printf("Received message for self: %s\n", msg[1])
			s.message <- msg[1]
		case "token":
			fmt.Printf("Received token message: %s\n", msg[1])
			s.requesteres = append(s.requesteres, msg[1])
		default:
			fmt.Printf("Received message from peer: %s\n", msg[0])
		}
	}
}

type Synchronized interface {
	Execute(cond Conditinal) error
}

type Conditinal interface {
	Wait() error
	Signal() error
	SignalAll() error
}

func (m *Monitor) Execute(sync Synchronized) error {
	if sync == nil {
		return nil
	}
	if err := sync.Execute(m.cond); err != nil {
		return err
	}
	return nil
}

func (m *Monitor) HandleToken(who string) error {	
	m.mutex.Lock()
	if !m.lockAcquired {	
		m.tokenOwner = false
		m.mutex.Unlock()
		_, err := m.Pub.Send(fmt.Sprintf("%d token %d", who, m.config.Self), 0)
		if err != nil {
			return fmt.Errorf("failed to send token request: %w", err)
		}
		msg := <- m.message
		if msg != "ok" {
			return fmt.Errorf("unexpected message received: %s", msg)
		}
		m.sub.SetUnsubscribe("token")
	}
	// m.requesteres = append(m.requesteres, msg)
}

func (m *Monitor) Lock() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.lockAcquired {
		return fmt.Errorf("lock already acquired")
	}
	if m.tokenOwner {
		m.lockAcquired = true
		return nil
	}
	m.mutex.Unlock()
	_, err := m.Pub.Send(fmt.Sprintf("token %d", m.config.Self), 0)
	if err != nil {
		return fmt.Errorf("failed to send token request: %w", err)
	}
	mes := <-m.message
	mesSplit := strings.Split(mes, " ")
	if len(mesSplit) < 2 {
		return fmt.Errorf("invalid message format received: %s", mes)
	}
	if mesSplit[0] != "token" {
		return fmt.Errorf("unexpected message received: %s", mes)
	}
	m.mutex.Lock()
	m.tokenOwner = true
	m.mutex.Unlock()
	m.sub.SetSubscribe("token")
	m.Pub.Send(fmt.Sprintf("%s ok", mesSplit[1]), 0)
	m.mutex.Lock()
	m.lockAcquired = true
	m.mutex.Unlock()
	return nil
}
