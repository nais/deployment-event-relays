PROTOC = $(shell which protoc)
PROTOC_GEN_GO = $(shell which protoc-gen-go)

.PHONY: deployment-event-relays

deployment-event-relays: 
	mkdir -p bin
	go build -o bin/deployment-event-relays cmd/deployment-event-relays/main.go

proto:
	wget -O event.proto https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --plugin=$(PROTOC_GEN_GO) --go_out=. event.proto
	mv event.pb.go pkg/deployment/
	rm -f event.proto

alpine:
	go build -a -installsuffix cgo -o deployment-event-relays cmd/deployment-event-relays/main.go

test:
	go test ./...
