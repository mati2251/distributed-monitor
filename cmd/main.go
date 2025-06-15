package main

import (
	"fmt"
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
	for m.Data.N != 0 {
		m.Wait()
	}
	m.Data.N = m.Data.N + m.Data.Aw
	for range 5 {
		m.Data.Buf = append(m.Data.Buf, m.Data.C)
		m.Data.C++
		fmt.Println("Writer, value:", m.Data.Buf[len(m.Data.Buf)-1])
	}
	time.Sleep(100 * time.Millisecond)
	m.Data.N = m.Data.N - m.Data.Aw
	m.SignalAll()
	return nil
}

func reader(m *rem.Monitor[*ReaderWriterData]) error {
	for m.Data.N&m.Data.Aw == m.Data.Aw || len(m.Data.Buf) == 0 {
		m.Wait()
	}
	m.Data.N = m.Data.N + m.Data.Ar
	fmt.Println("Reader, value:", m.Data.Buf[0])
	m.Data.Buf = m.Data.Buf[1:]
	time.Sleep(100 * time.Millisecond)
	m.Data.N = m.Data.N - m.Data.Ar
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
		fmt.Printf("failed to create monitor: %s\n", err.Error())
		return nil, err
	}
	rw := &ReaderWriter{
		monitor: monitor,
	}
	return rw, nil
}

func main() {
	time.Sleep(2 * time.Second)
	rw, err := NewReaderWriter()
	if err != nil {
		fmt.Printf("failed to create ReaderWriter: %s\n", err.Error())
		return
	}

	go rw.Writer()

	time.Sleep(10 * time.Second)
}
