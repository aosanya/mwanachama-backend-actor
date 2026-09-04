# QA

Test coverage and results for `mwanachama-backend-actor`. See this repo's
root for the actual test files (`*_test.go`); `go test ./...` runs the
in-memory-fake-backed unit suite with no database required, and
`postgres_integration_test.go` additionally exercises the real Postgres
wiring when `POSTGRES_URL` is set.
