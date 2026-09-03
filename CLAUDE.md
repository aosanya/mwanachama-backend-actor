# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-user

Extraction of `mwanachama-backend-api-gateway`'s `member` domain (and
`chapter`, renamed `Group`) onto the `mwanachama-backend-taskmanager`
pattern: domain logic AND storage both delegated to
`mwanachama-backend-shared/entitygraph.DataManager`, imported directly by
the gateway process — no separate service, no gRPC, no proto. Module path
`github.com/aosanya/mwanachama-backend-user`.

Decided in a dev-research session, 2026-09-03. Full decision record and the
Q&A trail: `mwanachama-backend-api-gateway/documentation/2. design/todo.md`,
rows DSN-1698 (decided) and DSN-1699 (four gaps the session identified but
did not settle).

## Porting notes

- `user.go`'s `UserManager` interface and `models.go`'s domain types
  (`Member`, `Group`, `Registration`) port the gateway's
  `internal/domain/member.Member`/`.Registration` and
  `internal/domain/chapter.Chapter` field-for-field, with two changes:
  `Chapter` → `Group` (DSN-1698 decision 3 — the gateway is responsible for
  presenting it under an org-configurable label; this repo never hardcodes
  the word "Chapter" anywhere a type_id belongs), and `Registration` becomes
  the `registered_at` relationship, not a struct backed by a table.
- `schema.go`'s `DefaultUserSchema(instance)` declares two types, `Member`
  and `Group`, connected by `registered_at` / `has_member`. `instance` names
  the gateway's mounted copy (e.g. `"member"`) and only steers the two
  types' `StorageCollection` labels, kept in step with whatever Postgres
  table prefix the gateway wires alongside it — the gateway may mount this
  package more than once, each with its own instance name/table prefix/route
  prefix. Timestamps are
  strings (RFC 3339), matching `mwanachama-backend-taskmanager`'s own
  convention for `entitygraph`-stored entities.
- **`Hierarchy` and `Level` did NOT move here.** DSN-1699 gap 1 was left
  unresolved by the design session; this repo's first build took the
  DEFAULT the design doc names and recorded it explicitly (see `schema.go`'s
  package doc and `documentation/2. design/README.md`): `hierarchy` and
  `chapter_level` stay physically in the gateway's own Postgres tables.
  `Group` carries `hierarchy_id` / `level_id` / `parent_id` /
  `anchor_level_override` as plain string properties with **no foreign key**
  back to those gateway tables — and therefore no equivalent of the
  gateway's migration-000034 composite-FK guarantee that an anchor override
  names a level from the chapter's own hierarchy. If gap 1 is ever resolved
  the other way (hierarchy/level move into this repo as their own entity
  types), `MoveGroup`'s in-Go cycle walk and the property-only fields here
  both need revisiting.
- **`Registration.IsHome` carries no exclusivity.** DSN-1698 decision 8
  drops "one home chapter per member" outright, not relocated — do not
  reintroduce a uniqueness check on `is_home`, in this repo or in a caller.
- **`MoveGroup`'s cycle/subtree check is a Go walk, not a recursive CTE.**
  The gateway's Postgres `chapter_store.go` used `WITH RECURSIVE`;
  `entitygraph.DataManager` has no arbitrary-SQL escape hatch, so
  `group_impl.go` lists every group and walks the parent chain in memory.
  Fine at org scale; revisit if a deployment's group count ever makes this
  a real cost.
- **No act-log writing here, by design.** The gateway's
  `member.Repository.Deregister` composes and writes a
  `custody.ChapterActLogEntry` in the same transaction as the delete
  (DEV-1343/DEV-1344). This repo's `UserManager.Deregister` only removes the
  `registered_at` edge and returns the prior `Registration` (which still
  carries `IsHome` — the one fact a caller needs to compose the same act
  after the fact). `custody` is a gateway-internal domain this repo must not
  depend on; the gateway adapter (`internal/store/entitygraph` there) is
  where the act gets composed and written, in its own, separate call — not
  inside the same Postgres transaction as the edge delete, since the two
  now live behind different interfaces even though they happen to share one
  physical database. This is a deliberate, noted loss of atomicity, not an
  oversight.
- **Member search (the console directory, DEV-1034) is NOT built here.**
  `member.Searcher.Search` composes member + chapter + role + verification
  data, and role/verification stay gateway-side. The gateway already has a
  working composer for exactly this shape —
  `internal/store/memory/member_search.go`'s `MemberSearchService`, which
  depends only on the gateway's own domain interfaces (`member.Repository`,
  `chapter.Repository`, `role.Repository`, `verification.Repository`) and
  is therefore reusable unchanged, wired against the new
  `internal/store/entitygraph` adapters instead of `memory`'s own stores.
  Nothing in this repo needs to reproduce that composition.
- Straightforward CRUD files (`member_impl.go`, `group_impl.go`,
  `registration_impl.go`, `converters.go`) only ever call
  `entitygraph.DataManager` — port with minimal churn, matching
  `mwanachama-backend-taskmanager`'s own porting note for its equivalent
  files.

## Conventions

- Task status lives on
  [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md).
- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- Before wiring into `mwanachama-backend-api-gateway`, the two Go interfaces
  that must NOT change are `internal/domain/member.Repository`/`.Searcher`
  and `internal/domain/chapter.Repository` — route paths, request/response
  shapes and status codes all depend on those staying byte-for-byte
  identical. New adapter types satisfying them live in the gateway's
  `internal/store/entitygraph`, not here.
