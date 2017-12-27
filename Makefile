.PHONY: all
all: test

.PHONY: dependencies
dependencies:
	@echo "Installing Glide and locked dependencies..."
	glide --version || go get -u -f github.com/Masterminds/glide
	glide install

.PHONY: test
test: dependencies
	go test -race -v github.com/alphaqiu/EventEmitter/event