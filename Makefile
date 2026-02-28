BIN_DIR := /usr/local/bin

.PHONY: build install deploy

build:
	go build -tags "fts5" -o mc ./cmd/mc
	go build -tags "fts5" -o server ./cmd/server

install: build
	@test -L $(BIN_DIR)/mc || sudo ln -s $(CURDIR)/mc $(BIN_DIR)/mc
	@echo "mc installed → $(BIN_DIR)/mc"

deploy: install
	git pull --rebase
	git push
