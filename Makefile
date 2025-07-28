PLUGIN_NAME := rockvalues
VERSION := 0.9.0
DIST_DIR := dist

.PHONY: build package clean

# Build pour toutes les plateformes
build:
	@echo "Building $(PLUGIN_NAME) v$(VERSION)..."
	@mkdir -p $(DIST_DIR)/bin/linux
	@mkdir -p $(DIST_DIR)/bin/windows
	@cd go; GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ../$(DIST_DIR)/bin/linux/$(PLUGIN_NAME) 
	@cd go; GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ../$(DIST_DIR)/bin/windows/$(PLUGIN_NAME).exe

# Copier les fichiers nécessaires
package: build
	@echo "Packaging plugin"
	@cp plugin.yaml $(DIST_DIR)/
	@cp -r scripts $(DIST_DIR)/
	@cp LICENSE $(DIST_DIR)/

# Créer une archive
archive: package
	@echo "Creating archive..."
	@cd $(DIST_DIR) && tar -czf ../$(PLUGIN_NAME)-$(VERSION).tar.gz .
	@echo "Archive created: $(PLUGIN_NAME)-$(VERSION).tar.gz"

clean:
	@rm -rf $(DIST_DIR)
	@rm -f *.tar.gz

install: package
	@helm plugin install $(DIST_DIR)

uninstall:
	@helm plugin uninstall $(PLUGIN_NAME)

test:
	@echo "Testing plugin..."
