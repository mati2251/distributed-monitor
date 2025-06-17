package rem

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"slices"
	"strconv"
)

func (m *Monitor[T]) Run() {
	go func(m *Monitor[T]) {
		gob.Register(TokenRequest[T]{})
		gob.Register(Request{})
		gob.Register(m.Data)
		for {
			msg, err := m.sub.RecvMessage(0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error receiving message: %v\n", err)
				continue
			}
			if len(msg) > 1 {
				m.cond.L.Lock()
				if msg[0] == BRODCAST {
					m.handleRequest(msg[1])
				} else if msg[0] == SIGNAL {
					m.handleSignal(msg[1])
				} else if msg[0] == WAIT {
					m.handleWait(msg[1])
				} else {
					m.handleToken(msg[1])
				}
				m.cond.L.Unlock()
			}
		}

	}(m)
}

func (m *Monitor[T]) handleRequest(msgParts string) {
	var request Request
	decoder := gob.NewDecoder(bytes.NewBufferString(msgParts))
	err := decoder.Decode(&request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding request: %v\n", err)
		return
	}

	m.rn[request.Id] = max(m.rn[request.Id], request.Timestamp)
	if m.hasToken && m.config.Self != request.Id {
		if m.token.Lp[request.Id] < m.rn[request.Id] && !m.locked {
			m.token.Q = append(m.token.Q, request.Id)
			m.sendToken()
		}
	}
}

func (m *Monitor[T]) handleToken(msg string) {
	var tokenRequest TokenRequest[T]
	decoder := gob.NewDecoder(bytes.NewBufferString(msg))
	err := decoder.Decode(&tokenRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		fmt.Fprintf(os.Stderr, "Error decoding token request: %v\n", err)
		return
	}
	m.token = tokenRequest.Token
	m.Data = tokenRequest.Data
	m.hasToken = true
	m.cond.Signal()
}

func (m *Monitor[T]) sendToken() {
	if len(m.token.Q) == 0 {
		return
	}

	m.hasToken = false
	id := m.token.Q[0]
	m.token.Q = m.token.Q[1:]

	tokenRequest := TokenRequest[T]{
		Token: m.token,
		Data:  m.Data,
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(tokenRequest)
	if err != nil {
		fmt.Printf("Error encoding token with gob: %v\n", err)
		return
	}
	if _, err := m.pub.SendMessage(id, buf.Bytes(), 0); err != nil {
		fmt.Printf("Error sending token: %v\n", err)
		return
	}
}

func (m *Monitor[T]) sendRequest() {
	request := Request{
		Id:        m.config.Self,
		Timestamp: m.rn[m.config.Self],
	}
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(request)
	if err != nil {
		fmt.Printf("Error encoding request with gob: %v\n", err)
		return
	}
	if _, err := m.pub.SendMessage(BRODCAST, buf.Bytes(), 0); err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
}

func (m *Monitor[T]) handleWait(msg string) {
	id, err := strconv.Atoi(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting message to ID: %v\n", err)
		return
	}
	if !slices.Contains(m.waiting, id) {
		m.waiting = append(m.waiting, id)
	}
}

func (m *Monitor[T]) sendWait() {
	if _, err := m.pub.SendMessage(WAIT, strconv.Itoa(m.config.Self), 0); err != nil {
		fmt.Printf("Error sending wait signal: %v\n", err)
		return
	}
}

func (m *Monitor[T]) handleSignal(msg string) {
	id, err := strconv.Atoi(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting message to ID: %v\n", err)
		return
	}

	for i, waitingId := range m.waiting {
		if waitingId == id {
			m.waiting = slices.Delete(m.waiting, i, i+1)
			break
		}
	}
	if id == m.config.Self && m.wait {
		go func() {
			m.cond.L.Lock()
			m.wait = false
			m.plock()
			m.cond.Signal()
			m.cond.L.Unlock()
		}()
	}
}

func (m *Monitor[T]) sendSignal(all bool) {
	if len(m.waiting) == 0 {
		return
	}
	ids := m.waiting[0:1]
	if all {
		ids = m.waiting
	}
	for _, id := range ids {
		if _, err := m.pub.SendMessage(SIGNAL, strconv.Itoa(id), 0); err != nil {
			fmt.Printf("Error sending signal: %v\n", err)
			return
		}
	}
}
