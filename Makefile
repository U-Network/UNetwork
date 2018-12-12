UNAME = $(shell uname)

install:
	@echo "\n--> Installing the UNetwork TestNet\n"
	go install ./cmd/uuu
	@echo "\n\nUNetwork, the TestNet for UNetWork (UUU) has successfully installed!"

build:
	go build -o build/uuu ./cmd/uuu
	@echo "\n\nUNetwork, the TestNet for UNetWork (UUU) has successfully build!"


.PHONY: build install
