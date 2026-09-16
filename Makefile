VERSION := $(if $(VERSION),$(VERSION),"0.0.0-dev")

all: client server jquery

client: client.go version.go core/*
	go build -ldflags "-X main.AppVersion=$(VERSION)" -o dimmy client.go version.go

server: jquery server.go version.go http-handlers.go devices/* core/* html/*
	go build -ldflags "-X main.AppVersion=$(VERSION)" -o dimmyd server.go version.go http-handlers.go

clean:
	rm dimmy dimmyd html/assets/jquery.js
	rm -rf deb
	rm dimmy_*.deb

jquery:
	wget https://code.jquery.com/jquery-3.4.1.min.js -O html/assets/jquery.js

install:
	install -m 755 dimmy dimmyd /usr/bin/
	mkdir -p /etc/dimmy /usr/share/dimmy /var/lib/dimmy
	cp dimmyd.conf.yaml.example rules.conf.yaml.example dimmy.conf.example /etc/dimmy/
	cp -R html/* /usr/share/dimmy
	install -D -m 755 system/dimmyd.init /etc/init.d/dimmyd
	install -D -m 644 system/dimmyd.service /lib/systemd/system/dimmyd.service
	bash system/dimmy.postinst configure

test:
	go test ./devices
	go test ./core

ARCH := $(if $(ARCH),$(ARCH),$(shell dpkg --print-architecture))
deb: all
	rm -rf deb
	mkdir -p deb/dimmy/usr/bin
	mkdir -p deb/dimmy/etc/dimmy
	mkdir -p deb/dimmy/etc/init.d
	mkdir -p deb/dimmy/usr/share/dimmy
	mkdir -p deb/dimmy/lib/systemd/system
	mkdir -p deb/dimmy/var/lib/dimmy
	mkdir deb/dimmy/DEBIAN
	cp deb.control deb/dimmy/DEBIAN/control
	sed -i'' "s/__version__/$(VERSION)/" deb/dimmy/DEBIAN/control
	sed -i'' "s/__arch__/$(ARCH)/" deb/dimmy/DEBIAN/control
	cat deb/dimmy/DEBIAN/control
	cp dimmy deb/dimmy/usr/bin
	cp dimmyd deb/dimmy/usr/bin
	install -m 755 system/dimmyd.init deb/dimmy/etc/init.d/dimmyd
	install -m 644 system/dimmyd.service deb/dimmy/lib/systemd/system
	install -m 755 system/dimmy.postinst deb/dimmy/DEBIAN/postinst
	cp dimmyd.conf.yaml.example deb/dimmy/etc/dimmy/dimmyd.conf.yaml.example
	cp rules.conf.yaml.example deb/dimmy/etc/dimmy/rules.conf.yaml.example
	cp dimmy.conf.example deb/dimmy/etc/dimmy/dimmy.conf.example
	cp -R html/* deb/dimmy/usr/share/dimmy
	dpkg-deb -Zgzip --root-owner-group --build deb/dimmy
