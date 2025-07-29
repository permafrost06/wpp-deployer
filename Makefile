BINARY_NAME=wpp-deployer
BUILD_DIR=build
INSTALL_PATH=/usr/local/bin

.PHONY: all build clean install uninstall test

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installation completed. Run '$(BINARY_NAME) install' to set up the workspace."

uninstall:
	@echo "Removing $(BINARY_NAME) from $(INSTALL_PATH)..."
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstalled. Note: ~/.wpp-deployer directory was not removed."

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

help:
	@echo "Available targets:"
	@echo "  build     - Build the binary"
	@echo "  install   - Install the binary to $(INSTALL_PATH) (requires sudo)"
	@echo "  uninstall - Remove the binary from $(INSTALL_PATH) (requires sudo)"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run tests"
	@echo "  help      - Show this help message" 