CGO_LDFLAGS_ALLOW = "-I.*"
UNAME = $(shell uname)


install:
	@echo "\n--> Installing the UNetwork TestNet\n"
ifeq ($(UNAME), Linux)
	CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_LDFLAGS_ALLOW="$(CGO_LDFLAGS_ALLOW)" go install ./cmd/uuu
endif
ifeq ($(UNAME), Darwin)
	CGO_LDFLAGS_ALLOW="$(CGO_LDFLAGS_ALLOW)" go install ./cmd/uuu
endif
	@echo "\n\nUNetwork, the TestNet for UNetWork (UUU) has successfully installed!"


build:
ifeq ($(UNAME), Linux)
    CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_LDFLAGS_ALLOW="$(CGO_LDFLAGS_ALLOW)" go build -o build/uuu ./cmd/uuu
endif
ifeq ($(UNAME), Darwin)
    CGO_LDFLAGS_ALLOW="$(CGO_LDFLAGS_ALLOW)" go build -o build/uuu ./cmd/uuu
endif
