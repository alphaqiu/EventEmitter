.PHONY: all
all: test

.PHONY: dependencies
dependencies:
	@echo "Installing Glide and locked dependencies..."
	glide --version || go get -u -f github.com/Masterminds/glide
	go get -u -f github.com/mattn/goveralls
	glide install

.PHONY: test
test: dependencies
	go test -v -covermode=count -coverprofile=coverage.out github.com/alphaqiu/EventEmitter/event
	goveralls -coverprofile=coverage.out -service=travis-ci -repotoken $(COVERALLS_TOKEN)