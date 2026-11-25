.DEFAULT_GOAL := help

RUN_ADDR ?= localhost:8080
BASE_URL ?= http://localhost:8080

.PHONY: help build run fmt test vet check

help:
	@printf '%s\n' \
		'make run    — запустить сервис (RUN_ADDR=localhost:8080, BASE_URL=http://localhost:8080)' \
		'make build  — собрать cmd/shortener/shortener' \
		'make fmt    — отформатировать Go-файлы' \
		'make test   — проверить пакеты и запустить существующие тесты' \
		'make vet    — выполнить статический анализ' \
		'make check  — выполнить build, test и vet' \
		'' \
		'Параметры запуска:' \
		'  make run RUN_ADDR=localhost:8888' \
		'  make run BASE_URL=http://localhost:8000' \
		'  make run RUN_ADDR=localhost:8888 BASE_URL=http://localhost:8000'

build:
	go build -o cmd/shortener/shortener ./cmd/shortener

run:
	go run ./cmd/shortener -a "$(RUN_ADDR)" -b "$(BASE_URL)"

fmt:
	gofmt -w .

test:
	go test ./...

vet:
	go vet ./...

check: build test vet
