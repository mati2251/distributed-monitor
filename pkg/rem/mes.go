package rem

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
)

func (m *Monitor[T]) Run() {
	go func(m *Monitor[T]) {
		gob.Register(TokenRequest[T]{})
		gob.Register(Request{})
		gob.Register(m.Data)
		for {
			fmt.Printf("Waiting for message...\n")
			msg, err := m.sub.RecvMessage(0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error receiving message: %v\n", err)
				continue
			}
			fmt.Printf("Received message %s\n", msg)
			if len(msg) > 1 {
				m.cond.L.Lock()
				if msg[0] != BRODCAST {
					m.handleToken(msg[1])
				} else {
					m.handleRequest(msg[1])
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
	fmt.Printf("Handling request with ID %d and timestamp %d\n", request.Id, request.Timestamp)
}

func (m *Monitor[T]) handleToken(msg string) {
	var tokenRequest TokenRequest[T]
	decoder := gob.NewDecoder(bytes.NewBufferString(msg))
	err := decoder.Decode(&tokenRequest)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding token request: %v\n", err)
		return
	}
	fmt.Printf("Handling token request with token %v and data %v\n", tokenRequest.Token, tokenRequest.Data)
}

func (m *Monitor[T]) sendToken() {
	tokenRequest := TokenRequest[T]{
		Token: m.token,
		Data:  m.Data,
	}

	if len(m.token.Q) == 0 {
		// TODO fix
		m.token.Q = append(m.token.Q, 2)
	}

	id := m.token.Q[0]
	m.token.Q = m.token.Q[1:]

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
	m.timestamp++

	request := Request{
		Id:        m.config.Self,
		Timestamp: m.timestamp,
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
	fmt.Printf("Sent request with ID %d and timestamp %d\n", request.Id, request.Timestamp)
}
