.PHONY: mod test lint build up down clean web

.NOTPARALLEL:

mod:
	go mod download

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint -v run ./...

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.serviceVersion=v0.0.0" -o .docker/.build/jigsaw.amd64 .

web:
	cd web && npm run build

up: web build
	docker compose up -d --build

down:
	docker compose down --remove-orphans

clean:
	docker rmi $(docker images -f dangling=true -q)
