.PHONY: deps clean build

deps:
	go get -u ./...

build:
	GOOS=linux GOARCH=amd64 go build -o register-user/build/main ./register-user/main.go
	GOOS=linux GOARCH=amd64 go build -o login/build/main ./login/main.go
