PROTOC = $(shell which protoc)
PROTOC_GEN_GO = $(shell which protoc-gen-go)

.PHONY: deploys2stdout deploys2influx deploys2vera bin proto

bin: deploys2stdout deploys2influx deploys2vera

deploys2stdout: cmd/deploys2stdout/main.go
	mkdir -p bin
	cd cmd/deploys2stdout && go build
	mv cmd/deploys2stdout/deploys2stdout bin/

deploys2influx: cmd/deploys2influx/main.go
	mkdir -p bin
	cd cmd/deploys2influx && go build
	mv cmd/deploys2influx/deploys2influx bin/

deploys2influx: cmd/deploys2vera/main.go
	mkdir -p bin
	cd cmd/deploys2vera && go build
	mv cmd/deploys2vera/deploys2vera bin/

proto:
	wget -O event.proto https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --plugin=$(PROTOC_GEN_GO) --go_out=. event.proto
	mv event.pb.go pkg/deployment/
	rm -f event.proto

alpine:
	go build -a -installsuffix cgo -o deploys2stdout cmd/deploys2stdout/main.go
	go build -a -installsuffix cgo -o deploys2influx cmd/deploys2influx/main.go
	go build -a -installsuffix cgo -o deploys2vera cmd/deploys2vera/main.go

test:
	go test ./...
