package main

import (
	"fmt"
	"time"

	"github.com/mati2251/distributed-monitor/pkg/rem"
)

func main() {
	config, error := rem.ReadConfig("config.json")
	if error != nil {
		fmt.Printf("failed to read config: %s\n", error.Error())

	}
	socket, err := rem.NewSocket(*config)
	if err != nil {
		fmt.Printf("failed to create socket: %s\n", err.Error())
		return
	}
	defer socket.Close()
	fmt.Println("Socket created successfully")
	time.Sleep(1 * time.Second)
	socket.Pub.Send("1 eloo", 0)
	socket.Loop()
}
