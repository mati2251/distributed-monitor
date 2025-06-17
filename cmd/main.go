package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mati2251/distributed-monitor/pkg/rem"
)

type ProducerConsumentData struct {
	Buf    []int
	GetPos int
	PutPos int
	N      int
	C      int
}

func producer(m *rem.Monitor[*ProducerConsumentData]) error {
	for ;m.Data.N == len(m.Data.Buf); {
		m.Wait()
	}
	m.Data.C ++
	m.Data.Buf[m.Data.PutPos] = m.Data.C
	m.Data.PutPos = (m.Data.PutPos + 1) % len(m.Data.Buf)
	fmt.Printf("Produced: %d\n", m.Data.C)
	m.Data.N ++
	m.SignalAll()
	return nil
}

func consument(m *rem.Monitor[*ProducerConsumentData]) error {
	for ;m.Data.N == 0; {
		m.Wait()
	}
	fmt.Printf("Consumed: %d\n", m.Data.Buf[m.Data.GetPos])
	m.Data.GetPos = (m.Data.GetPos + 1) % len(m.Data.Buf)
	m.Data.N --
	m.SignalAll()
	return nil
}

type ProducerConsument struct {
	monitor *rem.Monitor[*ProducerConsumentData]
}

func (rw *ProducerConsument) Consument() {
	rw.monitor.Synchronized(consument)
}

func (rw *ProducerConsument) Producer() {
	rw.monitor.Synchronized(producer)
}

func NewReaderWriter() (*ProducerConsument, error) {
	monitor, err := rem.NewMonitor(&ProducerConsumentData{
		Buf:    make([]int, 5),
		GetPos: 0,
		PutPos: 0,
		C:      0,
		N:      0,
	}, "config.json")
	monitor.Run()
	if err != nil {
		log.Printf("failed to create monitor: %s\n", err.Error())
		return nil, err
	}
	rw := &ProducerConsument{
		monitor: monitor,
	}
	return rw, nil
}

func main() {
	rw, err := NewReaderWriter()
	if err != nil {
		log.Printf("failed to create ReaderWriter: %s\n", err.Error())
		return
	}
	if len(os.Args) < 2 {
		log.Println("Please provide an argument: 'r' for reader or 'w' for writer")
		return
	}
	time.Sleep(1 * time.Second)
	arg := os.Args[1]
	randomSleep := time.Duration(100+rand.Intn(400)) * time.Millisecond
	for range 20 {
		if arg == "c" {
			rw.Consument()
		} else {
			rw.Producer()
		}
		time.Sleep(randomSleep)
	}
	time.Sleep(3 * time.Second)
}
