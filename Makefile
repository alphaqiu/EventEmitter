.PHONY: all
all: test

.PHONY: test
test:
	go test -race -v github.com/alphaqiu/EventEmitter/event