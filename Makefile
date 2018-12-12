UNAME = $(shell uname)

install:
    @echo "\n--> Installing the UNetwork TestNet\n"
    ifeq ($(UNAME), Linux)
    go install ./cmd/uuu
    endif
    ifeq ($(UNAME), Darwin)
    go install ./cmd/uuu
    endif
        @echo "\n\nUNetwork, the TestNet for UNetWork (UUU) has successfully installed!"

build:
    ifeq ($(UNAME), Linux)
    go build -o build/uuu ./cmd/uuu
    endif
    ifeq ($(UNAME), Darwin)
    go build -o build/uuu ./cmd/uuu
    endif
