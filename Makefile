PROTOC = $(shell which protoc)
PROTOC_GEN_GO = $(shell which protoc-gen-go)

.PHONY: deploys2stdout proto

deploys2stdout: cmd/deploys2stdout/main.go
	mkdir -p bin
	cd cmd/deploys2stdout && go build
	mv cmd/deploys2stdout/deploys2stdout bin/

proto:
	wget -O event.proto https://raw.githubusercontent.com/navikt/protos/master/deployment/event.proto
	$(PROTOC) --plugin=$(PROTOC_GEN_GO) --go_out=. event.proto
	mv event.pb.go pkg/deployment/
	rm -f event.proto
