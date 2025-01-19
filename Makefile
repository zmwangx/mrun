.PHONY: all test demo demo.gif

all:

test:
	go run ./cmd/test

demo:
	go run ./cmd/demo

demo.gif:
	vhs cmd/demo/demo.tape
