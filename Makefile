.PHONY: all

all: out/server out/ploker.wasm out/wasm_exec.js

clean:
	rm -Rf out/*

docker: all
	docker build -t knusbaum/ploker:latest .

publish: clean docker
	docker push knusbaum/ploker:latest

out/ploker.wasm: cmd/client/*
	GOOS=js GOARCH=wasm go build -o out/ploker.wasm ./cmd/client

out/wasm_exec.js:
	cp "$(shell go env GOROOT)/misc/wasm/wasm_exec.js" out/

out/server: cmd/server/*
	CGO_ENABLED=0 go build -o out/server ./cmd/server
