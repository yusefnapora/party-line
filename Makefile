
all: frontend backend

frontend:
	cd frontend && GOARCH=wasm GOOS=js go build -o ../web/app.wasm

backend:
	go build

.PHONY: frontend backend
