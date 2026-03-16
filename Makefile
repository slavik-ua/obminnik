include .env

BINARY_NAME=app
MAIN_PACKAGE_PATH=./cmd/api
BUILD_DIR=bin

build:
	@echo "Building..."
	go build -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

run: build
		${BUILD_DIR}/${BINARY_NAME}

test:
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -rf ${BUILD_DIR}
	go clean

MIGRATIONS_DIR=sql/migrations

migrate-up:
	goose -dir ${MIGRATIONS_DIR} postgres "${DB_URL}" up

migrate-down:
	goose -dir {MIGRATIONS_DIR} postgres "${DB_URL}" down

.PHONY: build run test clean, migrate-up, migrate-down