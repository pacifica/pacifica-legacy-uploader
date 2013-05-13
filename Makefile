VERSION=0.7.5

UNAME=$(shell uname)

all: build-all

include Makefile.${UNAME}

pacificauploader.spec: pacificauploader.spec.in
	sed "s/@VERSION@/$(VERSION)/g" < pacificauploader.spec.in > pacificauploader.spec

dist: clean pacificauploader.spec
	rm -f pacificauploader-$(VERSION)
	ln -s . pacificauploader-$(VERSION)
	tar --exclude '.svn' --exclude '*.tar.*' --exclude packages --exclude 'pacificauploader-*/pacificauploader-*' -zcvf pacificauploader-$(VERSION).tar.gz pacificauploader-$(VERSION)/*

go/src/pacificauploaderd/common/version.go: go/src/pacificauploaderd/common/version.go.in
	sed "s/@VERSION@/$(VERSION)/g" < go/src/pacificauploaderd/common/version.go.in > go/src/pacificauploaderd/common/version.go

rpm: dist
	mkdir -p packages/bin packages/src
	rpmbuild --define '_rpmdir '`pwd`'/packages/bin' --define '_srcrpmdir '`pwd`'/packages/src' -ta pacificauploader-$(VERSION).tar.gz

rpms: rpm

check-common: run-tests
	
build-tests:
	mkdir -p build
	cd build; go test -c archiver; \
		go test -c pacificauploaderd/common; \
		go test -c sqlite
	
clean:
	cd qmake && make clean || true
	rm -rf build
	rm -rf msisdk
	rm -rf go/bin go/pkg
	rm -f pacificauploader.wxs
	rm -f pacificauploaderui.wxs
	rm -f pacificauploadersdk.wxs
	rm -f pacificauploaderall.wxs
	rm -f pacificauploadergen.wxs
	rm -f pacificauploaderuigen.wxs
	rm -f pacificauploader*.wixobj
	rm -f pacificauploader*.wixpdb
	rm -f pacificauploader*.msi
	rm -f pacificauploader*.exe
	rm -f pacificauploader.spec
	rm -f pacificauploader-$(VERSION)
	rm -f pacificauploader-$(VERSION).tar.gz 
