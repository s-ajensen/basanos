.PHONY: build clean test schema install uninstall

BIN_DIR := bin
PREFIX := /usr/local
INSTALL_DIR := $(PREFIX)/bin
BINARIES := basanos assert_equals assert_contains assert_matches assert_gt assert_gte assert_lt assert_lte

build: $(addprefix $(BIN_DIR)/,$(BINARIES))

$(BIN_DIR)/basanos: main.go $(shell find internal -name '*.go')
	@mkdir -p $(BIN_DIR)
	go build -o $@ .

$(BIN_DIR)/assert_%: cmd/assert_%/main.go $(shell find internal -name '*.go')
	@mkdir -p $(BIN_DIR)
	go build -o $@ ./cmd/assert_$*

clean:
	rm -rf $(BIN_DIR)

test:
	go test ./... -count=1

schema:
	@mkdir -p schema
	go run ./cmd/gen-schema > schema/events.json

install: build
	@mkdir -p $(INSTALL_DIR)
	@for bin in $(BINARIES); do \
		cp $(BIN_DIR)/$$bin $(INSTALL_DIR)/$$bin; \
		chmod 755 $(INSTALL_DIR)/$$bin; \
	done
	@echo "Installed to $(INSTALL_DIR)"

uninstall:
	@for bin in $(BINARIES); do \
		rm -f $(INSTALL_DIR)/$$bin; \
	done
	@echo "Uninstalled from $(INSTALL_DIR)"
