noop:
	echo "no op"

generate:
	go get -v github.com/jteeuwen/go-bindata/go-bindata@6025e8de665b31fa74ab1a66f2cddd8c0abf887e
	go get -v github.com/elazarl/go-bindata-assetfs/...
	go generate -x cmd/server/server.go