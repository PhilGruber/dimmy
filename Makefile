
all: client server

client: client.go
	go build -o dimmy client.go

server: server.go device.go html/*
	go build -o dimmyd server.go device.go

install:
	cp dimmy /usr/bin
	cp dimmyd /usr/bin