VERSION := $(if $(VERSION),$(VERSION),"0.0.0-dev")

all: client server jquery

client: client.go core/*
	go build -ldflags "-X main.AppVersion=$(VERSION)" -o dimmy client.go

server: jquery server.go http-handlers.go devices/* core/* html/*
	go build -ldflags "-X main.AppVersion=$(VERSION)" -o dimmyd server.go http-handlers.go

clean:
	rm dimmy dimmyd html/assets/jquery.js
	rm -rf deb
	rm dimmy_*.deb

jquery:
	wget https://code.jquery.com/jquery-3.4.1.min.js -O html/assets/jquery.js

install:
	cp dimmy /usr/bin
	cp dimmyd /usr/bin
	VERSION -f /etc/dimmy/dimmyd.conf.yaml || cp dimmyd.conf.yaml.example /etc/dimmy/dimmyd.conf.yaml
	mkdir -p /usr/share/dimmy
	cp -R html/* /usr/share/dimmy

deb: all
	rm -rf deb
	mkdir -p deb/dimmy/usr/bin
	mkdir -p deb/dimmy/etc/dimmy
	mkdir -p deb/dimmy/usr/share/dimmy
	mkdir deb/dimmy/DEBIAN
	cp deb.control deb/dimmy/DEBIAN/control
	sed -i'' "s/__version__/$(VERSION)/" deb/dimmy/DEBIAN/control
	sed -i'' "s/__arch__/$(ARCH)/" deb/dimmy/DEBIAN/control
	cat deb/dimmy/DEBIAN/control
	cp dimmy deb/dimmy/usr/bin
	cp dimmyd deb/dimmy/usr/bin
	cp *.conf.yaml.example deb/dimmy/etc/dimmy/
	cp -R html/* deb/dimmy/usr/share/dimmy
	dpkg-deb -Zgzip --root-owner-group --build deb/dimmy
