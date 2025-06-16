package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mati2251/distributed-monitor/pkg/rem"
)

type ReaderWriterData struct {
	N   int
	C   int
	Ar  int
	Aw  int
	Buf []int
}

func writer(m *rem.Monitor[*ReaderWriterData]) error {
	log.Println("Writer wants to access to cs")
	log.Printf("Writer data %d\n", m.Data.N)
	for m.Data.N != 0 {
		m.Wait()
	}
	m.Data.N = m.Data.N + m.Data.Aw
	defer func() {
		m.Data.N = 0
	}()
	log.Printf("Writer data %d updated\n", m.Data.N)
	for range 5 {
		m.Data.Buf = append(m.Data.Buf, m.Data.C)
		m.Data.C++
		log.Println("Writer, value:", m.Data.Buf[len(m.Data.Buf)-1])
	}
	time.Sleep(500 * time.Millisecond)
	log.Printf("Writer data %d updated end\n", m.Data.N)
	m.SignalAll()
	return nil
}

func reader(m *rem.Monitor[*ReaderWriterData]) error {
	log.Println("Reader wants to access to cs")
	log.Printf("Reader data %d\n", m.Data.N)
	for m.Data.N&m.Data.Aw == m.Data.Aw || len(m.Data.Buf) == 0 {
		log.Printf("Reader data %d and length %d\n", m.Data.N, len(m.Data.Buf))
		m.Wait()
	}
	m.Data.N = m.Data.N + m.Data.Ar
	defer func() {
		m.Data.N = m.Data.N - m.Data.Ar
	}()
	log.Printf("Reader data %d updated\n", m.Data.N)
	for range 5 {
		if len(m.Data.Buf) == 0 {
			log.Println("Reader, buffer is empty")
			return nil
		}
		log.Println("Reader, value:", m.Data.Buf[0])
		m.Data.Buf = m.Data.Buf[1:]
	}
	time.Sleep(500 * time.Millisecond)
	log.Printf("Reader data %d updated end\n", m.Data.N)
	m.SignalAll()
	return nil
}

type ReaderWriter struct {
	monitor *rem.Monitor[*ReaderWriterData]
}

func (rw *ReaderWriter) Reader() {
	rw.monitor.Synchronized(reader)
}

func (rw *ReaderWriter) Writer() {
	rw.monitor.Synchronized(writer)
}

func NewReaderWriter() (*ReaderWriter, error) {
	monitor, err := rem.NewMonitor(&ReaderWriterData{
		N:   0,
		C:   0,
		Ar:  2,
		Aw:  1,
		Buf: make([]int, 0),
	}, "config.json")
	monitor.Run()
	if err != nil {
		log.Printf("failed to create monitor: %s\n", err.Error())
		return nil, err
	}
	rw := &ReaderWriter{
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
	for range 5 {
		if arg == "r" {
			rw.Reader()
		} else {
			rw.Writer()
		}
		time.Sleep(randomSleep)
	}
	time.Sleep(2 * time.Second)
}
