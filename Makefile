PROTOC = $(shell which protoc)
PROTOC_GEN_GO = $(shell which protoc-gen-go)

.PHONY: deployment-event-relays

deployment-event-relays: 
	mkdir -p bin
	go build -o bin/deployment-event-relays cmd/deployment-event-relays/main.go

proto:
	wget -O pkg/deployment/event.proto https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --go_opt=Mpkg/deployment/event.proto=github.com/nais/deployment-event-relays/pkg/deployment,paths=source_relative --go_out=. pkg/deployment/event.proto
	rm -f pkg/deployment/event.proto

alpine:
	mkdir -p bin
	go build -a -installsuffix cgo -o bin/deployment-event-relays cmd/deployment-event-relays/main.go

test:
	go test ./...
