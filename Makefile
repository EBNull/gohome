.PHONY: build
build:
	goreleaser build --single-target --snapshot --clean

.PHONY: release
release:
	goreleaser release --clean

