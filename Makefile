
all: client server

client: client.go
	go build -o dimmy client.go

server: server.go device.go html/*
	go build -o dimmyd server.go device.go

jquery:
	wget https://code.jquery.com/jquery-3.4.1.min.js -o assets/jquery.js

install:
	cp dimmy /usr/bin
	cp dimmyd /usr/bin
