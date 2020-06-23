version="1.0.3"
version_file=VERSION
working_dir=$(shell pwd)
arch="armhf"

clean:
	-rm ./src/sensibo
	find . -name '.DS_Store' -type f -delete

build-go:
	go build -o sensibo src/service.go

build-go-arm:
	cd ./src;GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o sensibo service.go;cd ../

build-go-amd:
	GOOS=linux GOARCH=amd64 go build -o sensibo src/service.go


configure-arm:
	python ./scripts/config_env.py prod $(version) armhf

configure-amd64:
	python ./scripts/config_env.py prod $(version) amd64


package-tar:
	tar cvzf sensibo_$(version).tar.gz sensibo VERSION


package-deb-doc:
	@echo "Packaging application as debian package"
	chmod a+x package/debian/DEBIAN/*
	mkdir -p package/debian/var/log/thingsplex/sensibo package/debian/var/lib/futurehome/sensibo/data package debian/usr/bin
	mkdir -p package/build
	cp ./src/sensibo package/debian/opt/thingsplex/sensibo
	cp VERSION package/debian/var/lib/futurehome/sensibo
	docker run --rm -v ${working_dir}:/build -w /build --name debuild debian dpkg-deb --build package/debian
	@echo "Done"


tar-arm: build-js build-go-arm package-deb-doc
	@echo "The application was packaged into tar archive "

deb-arm : clean configure-arm build-go-arm package-deb-doc
	mv package/debian.deb package/build/sensibo_$(version)_armhf.deb

deb-amd : configure-amd64 build-go-amd package-deb-doc
	mv debian.deb sensibo_$(version)_amd64.deb

run :
	cd ./src; go run service.go -c testdata ../


.phony : clean
