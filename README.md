# mwanachama-backend-actor

Members and the groups (chapters) they belong to: who is in the network, which group they are assigned to, their roles, and the group hierarchy.

A Go library backed by Postgres through GORM. It is imported directly by the
Mwanachama API gateway; there is no separate service to run.

## Use

```sh
go get github.com/aosanya/mwanachama-backend-actor
```

`Migrate(db, tables)` creates the tables, and `NewUserManager(db, tables)` returns the manager
that the rest of your code calls. The `routes/` package builds the HTTP
handlers for the surface that needs no gateway-specific wiring.

## Test

```sh
go test ./...
```

Unit tests run against in-memory SQLite. `postgres_integration_test.go` runs
against a real Postgres only when `POSTGRES_URL` is set.

## Licence

Apache-2.0. See [LICENSE](LICENSE). Design notes and the task board are in
[documentation/](documentation/).
