
all: client server jquery

client: client.go api.go
	go build -o dimmy client.go api.go

server: jquery server.go device.go api.go html/*
	go build -o dimmyd server.go device.go api.go sensor.go dimmable.go light.go plug.go thermostat.go switch.go zsensor.go zlight.go zigbee2mqtt.go group.go

clean:
	rm dimmy dimmyd html/assets/jquery.js

jquery:
	wget https://code.jquery.com/jquery-3.4.1.min.js -O html/assets/jquery.js

install:
	cp dimmy /usr/bin
	cp dimmyd /usr/bin
	test -f /etc/dimmyd.conf || cp dimmyd.conf.example /etc/dimmyd.conf
	mkdir -p /usr/share/dimmy
	cp -R html/* /usr/share/dimmy
