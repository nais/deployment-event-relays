.PHONY: deploys2stdout

deploys2stdout: cmd/deploys2stdout/main.go
	mkdir -p bin
	cd cmd/deploys2stdout && go build
	mv cmd/deploys2stdout/deploys2stdout bin/
