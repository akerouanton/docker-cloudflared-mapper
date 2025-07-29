PLUGIN_TAG ?= latest
PLUGIN_NAME = albinkerouanton006/cloudflared-mapper:$(PLUGIN_TAG)

.PHONY: build
build:
	docker build --target rootfs --output=type=local,dest=./plugin/rootfs .
	docker plugin disable $(PLUGIN_NAME) || true
	docker plugin rm -f $(PLUGIN_NAME) || true
	docker plugin create $(PLUGIN_NAME) ./plugin

.PHONY: push
push:
	docker plugin push $(PLUGIN_NAME)

.PHONY: install
install:
	docker plugin install --alias cloudflared $(PLUGIN_NAME)
