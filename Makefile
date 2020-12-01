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

	GOOS=linux GOARCH=amd64 go build -o get-category-product/build/main ./get-category-product/main.go
	GOOS=linux GOARCH=amd64 go build -o get-cp/build/main ./get-cp/main.go
	GOOS=linux GOARCH=amd64 go build -o get-markets/build/main ./get-markets/main.go
	GOOS=linux GOARCH=amd64 go build -o get-associated-tenants/build/main ./get-associated-tenants/main.go
	GOOS=linux GOARCH=amd64 go build -o get-markets-register/build/main ./get-markets-register/main.go

	GOOS=linux GOARCH=amd64 go build -o get-products-by-category-product/build/main ./get-products-by-category-product/main.go
	GOOS=linux GOARCH=amd64 go build -o create-order/build/main ./create-order/main.go
	GOOS=linux GOARCH=amd64 go build -o list-orders-clients/build/main ./list-orders-clients/main.go
	GOOS=linux GOARCH=amd64 go build -o list-order-tenant/build/main ./list-order-tenant/main.go