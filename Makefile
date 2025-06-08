ID=1

.PHONY: run
run:
	SELF=$(ID) go run cmd/main.go

.PHONY: tun
tun:
	sudo ip tuntap add dev tun0 mode tun
	sudo ip tuntap add dev tun1 mode tun
	sudo ip tuntap add dev tun2 mode tun
	sudo ip tuntap add dev tun3 mode tun

	sudo ip addr add 10.0.0.1/24 dev tun0
	sudo ip addr add 10.0.0.2/24 dev tun1
	sudo ip addr add 10.0.0.3/24 dev tun2
	sudo ip addr add 10.0.0.4/24 dev tun3

	sudo ip link set dev tun0 up
	sudo ip link set dev tun1 up
	sudo ip link set dev tun2 up
	sudo ip link set dev tun3 up
