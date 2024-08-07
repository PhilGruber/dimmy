
all: client server jquery

client: client.go core/*
	go build -o dimmy client.go

server: jquery server.go devices/* core/* html/*
	go build -o dimmyd server.go

clean:
	rm dimmy dimmyd html/assets/jquery.js

jquery:
	wget https://code.jquery.com/jquery-3.4.1.min.js -O html/assets/jquery.js

install:
	cp dimmy /usr/bin
	cp dimmyd /usr/bin
	test -f /etc/dimmyd.conf.yaml || cp dimmyd.conf.yaml.example /etc/dimmyd.conf.yaml
	mkdir -p /usr/share/dimmy
	cp -R html/* /usr/share/dimmy
