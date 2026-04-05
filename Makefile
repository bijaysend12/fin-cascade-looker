.PHONY: dev build clean frontend

dev:
	go run main.go

frontend:
	cd frontend && npm install && npm run build

build: frontend
	go build -o bin/looker .

clean:
	rm -rf bin/ frontend/dist/ frontend/node_modules/
