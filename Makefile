.PHONY: build test test-pg vet clean ddl

## Verify the module compiles cleanly.
build:
	go build ./...

## Unit tests (sqlite; the Postgres test skips itself without POSTGRES_URL).
test:
	go test ./...

## The same suite with the Postgres test actually running.
## Expects POSTGRES_URL. Unlike catalog, this repo carries no build tag —
## postgres_integration_test.go skips itself when the variable is unset.
test-pg:
	@test -n "$(POSTGRES_URL)" || (echo "POSTGRES_URL is not set, so the Postgres test would skip itself." && exit 1)
	go test ./...

vet:
	go vet ./...

## Print the DDL the shipped domain spec would run, for review.
ddl:
	go run ./cmd/ddl "spec/examples/agency.actor.json" postgres

clean:
	rm -rf bin/
