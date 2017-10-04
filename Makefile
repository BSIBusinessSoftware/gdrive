all: build        

VER = $(shell git describe --tags)
	

.PHONY: build
build:
	go build -ldflags "-X main.Version=$(VER)"
