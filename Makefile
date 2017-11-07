all: build        

VER = $(shell git describe --tags)
BUILDDATE=$(shell date '+%Y/%m/%d %H:%M:%S %Z')	

LDFLAGS=-ldflags "-X main.Version=$(VER) -X \"main.BuildDate=$(BUILDDATE)\""

.PHONY: build install
build:
	go build -x $(LDFLAGS)

install:
	go install -x $(LDFLAGS)
