package main

import (
	"fmt"
	"time"

	"github.com/mati2251/distributed-monitor/pkg/rem"
)

type ReaderWriterData struct {
	n   int
	c   int
	ar  int
	aw  int
	buf []int
}

func writer(c *rem.Cond, rw *ReaderWriterData) error {
	for rw.n != 0 {
		c.Wait()
	}
	rw.n = rw.n + rw.aw
	for range 5 {
		rw.buf = append(rw.buf, rw.c)
		rw.c++
		fmt.Println("Writer, value:", rw.buf[len(rw.buf)-1])
	}
	time.Sleep(100 * time.Millisecond)
	rw.n = rw.n - rw.aw
	c.SignalAll()
	return nil
}

func reader(c *rem.Cond, rw *ReaderWriterData) error {
	for rw.n&rw.aw == rw.aw || len(rw.buf) == 0 {
		c.Wait()
	}
	rw.n = rw.n + rw.ar
	fmt.Println("Reader, value:", rw.buf[0])
	rw.buf = rw.buf[1:]
	time.Sleep(100 * time.Millisecond)
	rw.n = rw.n - rw.ar
	c.SignalAll()
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
		n:   0,
		c:   0,
		ar:  2,
		aw:  1,
		buf: make([]int, 0),
	})
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
	rw, err := NewReaderWriter()
	if err != nil {
		fmt.Printf("failed to create ReaderWriter: %s\n", err.Error())
		return
	}

	for range 25 {
		go rw.Reader()
	}

	for range 5 {
		go rw.Writer()
	}

	time.Sleep(10 * time.Second)
}
