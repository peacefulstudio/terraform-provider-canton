.PHONY: build test test-race install-dev uninstall-dev clean

BINARY      := terraform-provider-canton
DEV_TFRC    := $(CURDIR)/.dev.tfrc
PROVIDER    := peacefulstudio/canton

build:
	go build -o $(BINARY) .

test:
	go test ./...

test-race:
	go test -race ./...

install-dev: build
	@printf 'provider_installation {\n  dev_overrides {\n    "$(PROVIDER)" = "$(CURDIR)"\n  }\n  direct {}\n}\n' > $(DEV_TFRC)
	@echo
	@echo "Built ./$(BINARY) at $(CURDIR)"
	@echo
	@echo "Activate dev override for THIS shell:"
	@echo "    export TF_CLI_CONFIG_FILE=$(DEV_TFRC)"
	@echo
	@echo "Or add the snippet from $(DEV_TFRC) to your ~/.terraformrc."
	@echo "While active, Terraform ignores lockfiles and version constraints"
	@echo "for $(PROVIDER) and prints a warning on every run."

uninstall-dev:
	rm -f $(DEV_TFRC)
	@echo "Removed $(DEV_TFRC) — unset TF_CLI_CONFIG_FILE in your shell to deactivate."

clean:
	rm -f $(BINARY) $(DEV_TFRC) coverage.out
