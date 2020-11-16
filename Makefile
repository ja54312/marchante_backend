.PHONY: deps clean build

deps:
	go get -u ./...

build:
	GOOS=linux GOARCH=amd64 go build -o register-user/build/main ./register-user/main.go
	GOOS=linux GOARCH=amd64 go build -o login/build/main ./login/main.go

	GOOS=linux GOARCH=amd64 go build -o add-products/build/main ./add-products/main.go
	GOOS=linux GOARCH=amd64 go build -o get-products/build/main ./get-products/main.go
	GOOS=linux GOARCH=amd64 go build -o update-products/build/main ./update-products/main.go
	GOOS=linux GOARCH=amd64 go build -o disabled-product/build/main ./disabled-product/main.go

	GOOS=linux GOARCH=amd64 go build -o forgot-pass/build/main ./forgot-pass/main.go
	GOOS=linux GOARCH=amd64 go build -o change-pass/build/main ./change-pass/main.go
