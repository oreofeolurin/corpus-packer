.PHONY: test install coverage

THRESHOLD ?= $(or $(COVERAGE_THRESHOLD),60)

install:
	go get -t -v ./...

test: install
	go test -race -v ./... -coverprofile=cover.out

coverage:
	@total_coverage=$$(go tool cover -func=cover.out | grep total: | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$total_coverage >= $(THRESHOLD)" | bc) -eq 1 ]; then \
		echo "Coverage ($$total_coverage%) is above the threshold ($(THRESHOLD)%)"; \
	else \
		echo "Coverage ($$total_coverage%) is below the threshold ($(THRESHOLD)%)"; \
		exit 1; \
	fi