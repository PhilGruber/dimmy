VERSION=0.2.0
ARCH=amd64

all: client server jquery

client: client.go core/*
	go build -o dimmy client.go

server: jquery server.go devices/* core/* html/*
	go build -o dimmyd server.go

clean:
	rm dimmy dimmyd html/assets/jquery.js
	rm -rf deb

jquery:
	wget https://code.jquery.com/jquery-3.4.1.min.js -O html/assets/jquery.js

install:
	cp dimmy /usr/bin
	cp dimmyd /usr/bin
	test -f /etc/dimmy/dimmyd.conf.yaml || cp dimmyd.conf.yaml.example /etc/dimmy/dimmyd.conf.yaml
	mkdir -p /usr/share/dimmy
	cp -R html/* /usr/share/dimmy

deb: all
	rm -rf deb
	mkdir -p deb/dimmy/usr/bin
	mkdir -p deb/dimmy/etc/dimmy
	mkdir -p deb/dimmy/usr/share/dimmy
	mkdir deb/dimmy/DEBIAN
	cp deb.control deb/dimmy/DEBIAN/control
	sed -i "s/\$version/$(VERSION)/" deb/dimmy/DEBIAN/control
	cp dimmy deb/dimmy/usr/bin
	cp dimmyd deb/dimmy/usr/bin
	cp dimmyd.conf.yaml.example deb/dimmy/etc/dimmy/dimmyd.conf.yaml
	cp -R html/* deb/dimmy/usr/share/dimmy
	dpkg-deb --build deb/dimmy
	mv deb/dimmy.deb dimmy_$(VERSION)_$(ARCH).deb
