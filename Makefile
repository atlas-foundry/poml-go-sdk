.DEFAULT_GOAL := help

UPSTREAM_REPO ?= poml-lang/poml
TAG ?=

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make sync-upstream TAG=vX.Y.Z   # Sync vendored upstream assets and run parity tests"

.PHONY: sync-upstream
sync-upstream:
	@if [ -z "$(TAG)" ]; then \
		echo "TAG is required. e.g. make sync-upstream TAG=v0.5.0"; \
		exit 1; \
	fi
	UPSTREAM_REPO=$(UPSTREAM_REPO) TAG=$(TAG) bash scripts/sync_upstream.sh
