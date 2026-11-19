.DEFAULT_GOAL := help

.PHONY: help build run fmt test vet check

help:
	@printf '%s\n' \
		'make run    — запустить сервис на localhost:8080' \
		'make build  — собрать cmd/shortener/shortener' \
		'make fmt    — отформатировать Go-файлы' \
		'make test   — проверить пакеты и запустить существующие тесты' \
		'make vet    — выполнить статический анализ' \
		'make check  — выполнить build, test и vet'

build:
	go build -o cmd/shortener/shortener ./cmd/shortener

run:
	go run ./cmd/shortener

fmt:
	gofmt -w .

test:
	go test ./...

vet:
	go vet ./...

check: build test vet
