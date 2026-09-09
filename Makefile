.PHONY: build install-python install installer clean run test vet lint cover cover-check startup-bench check

BINARY_NAME=spectre
COVER_PROFILE=coverage.out
COVER_MIN=35

build:
	go build -o $(BINARY_NAME).exe cmd/spectre/main.go

installer:
	powershell -ExecutionPolicy Bypass -File ./scripts/build_installer.ps1

install-python:
	@echo "Installing Python dependencies..."
	@if [ -f analyzer/requirements.txt ]; then \
		pip install -r analyzer/requirements.txt; \
	else \
		echo "Warning: analyzer/requirements.txt not found. Skipping Python setup."; \
	fi

install: build install-python

clean:
	rm -rf $(BINARY_NAME) $(BINARY_NAME).exe bin dist spectre-installer.exe $(COVER_PROFILE) coverage coverage.xml .coverage .pytest_cache evidence_storage startup_perf.json *.db*

run: build
	./$(BINARY_NAME)

test:
	go test ./...
	@if [ -d .venv ]; then .\.venv\Scripts\python.exe -m pytest; else python -m pytest; fi

vet:
	go vet ./...

lint: vet

cover:
	go test -coverprofile=$(COVER_PROFILE) ./...
	go tool cover -func=$(COVER_PROFILE)

cover-check:
	go test -coverprofile=$(COVER_PROFILE) ./...
	go run ./scripts/check_coverage.go -profile $(COVER_PROFILE) -min $(COVER_MIN)

startup-bench: build
	go run ./scripts/startup_perf -binary ./$(BINARY_NAME) -runs 20

check: lint test cover-check
