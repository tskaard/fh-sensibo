version_file=VERSION
working_dir=$(shell pwd)
arch="armhf"
version:=`git describe --tags | cut -c 2-`
remote_host = "fh@cube.local"

clean:
	-rm ./src/sensibo

init:
	git config core.hooksPath .githooks

build-go:
	go build -o sensibo src/service.go

build-go-arm: init
	cd ./src;GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o sensibo service.go;cd ../

build-go-amd: init
	GOOS=linux GOARCH=amd64 go build -o sensibo src/service.go


configure-arm:
	python ./scripts/config_env.py prod $(version) armhf

configure-amd64:
	python ./scripts/config_env.py prod $(version) amd64


package-tar:
	tar cvzf sensibo_$(version).tar.gz sensibo $(version_file)

clean-deb:
	find package -name ".DS_Store" -delete
	find package -name "delete_me" -delete


package-deb-doc: clean-deb
	@echo "Packaging application as debian package"
	chmod a+x package/debian/DEBIAN/*
	mkdir -p package/debian/var/log/thingsplex/sensibo package/debian/usr/bin
	mkdir -p package/build
	cp ./src/sensibo package/debian/opt/thingsplex/sensibo
	docker run --rm -v ${working_dir}:/build -w /build --name debuild debian dpkg-deb --build package/debian
	@echo "Done"


deb-arm : clean configure-arm build-go-arm package-deb-doc
	@echo "Building Futurehome ARM package"
	mv package/debian.deb package/build/sensibo_$(version)_armhf.deb
	@echo "Created package/build/sensibo_$(version)_armhf.deb"


deb-amd : configure-amd64 build-go-amd package-deb-doc
	@echo "Building Thingsplex AMD package"
	mv debian.deb package/build/sensibo_$(version)_amd64.deb

upload :
	scp package/build/sensibo_$(version)_armhf.deb $(remote_host):~/

remote-install : upload
	ssh -t $(remote_host) "sudo dpkg -i sensibo_$(version)_armhf.deb"

deb-remote-install : deb-arm remote-install
	@echo "Installed on remote host"

run :
	cd ./src; go run service.go -c testdata ../

.phony : clean
