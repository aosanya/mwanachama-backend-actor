# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-actor

Extraction of `mwanachama-backend-api-gateway`'s `member` domain (renamed
`Actor` — the gateway's own domain, routes, and tables keep the word
`member`; only this package's public naming changed) and `chapter` domain
(renamed `Group`): domain logic AND storage both live in this package,
imported directly by the gateway process — no separate service, no gRPC, no
proto. Module path `github.com/aosanya/mwanachama-backend-actor`.

Decided in a dev-research session, 2026-09-03. Full decision record and the
Q&A trail: `mwanachama-backend-api-gateway/documentation/2. design/todo.md`,
rows DSN-1698 (decided) and DSN-1699 (four gaps the session identified but
did not settle).

**Storage moved from entitygraph to GORM, 2026-09-04.** This package
originally stored Actor/Group/Registration through
`mwanachama-backend-shared/entitygraph.DataManager` — a generic,
runtime-versioned entity/relationship graph engine. The org has since moved
away from that graph-storage model in favor of fixed Go structs mapped to
plain relational tables, so this repo now uses GORM directly (chosen over
sqlc/ent given the domain here is small — three types). This was a
storage-layer swap only: `UserManager`'s method set is unchanged; only its
constructor (`NewUserManager`) and everything behind it changed shape. See
the `gormstore/` package.

**Domain types and GORM plumbing each live in their own subpackage,
2026-09-04.** `models/` holds the domain types (`Actor`, `Group`,
`GroupEdit`, `Registration`) one file per type; `gormstore/` holds every
GORM-specific piece (row structs, row↔domain conversion, `Migrate`), also
one file per entity. `tables.go` wraps `gormstore.DefaultTableNames`/
`gormstore.Migrate` — `mwanachamaactor.DefaultTableNames`,
`mwanachamaactor.Migrate` stay unchanged for every caller. `user.go`,
`actor_impl.go`, `group_impl.go`, `registration_impl.go`, and `errors.go`
stay at the root — they share the private `userManager` struct and the one
`UserManager` interface, which Go requires to live in a single package;
only files that are genuinely standalone (domain shapes, storage shapes)
became subpackages.

**No alias re-export for the domain types, 2026-09-04 (same day,
follow-up).** The root package originally re-exported `models.Actor`,
`models.Group`, `models.GroupEdit`, `models.Registration` as
`mwanachamaactor.Actor` etc. (`type Actor = models.Actor`, in a since-deleted
`models.go`) so every caller's import path stayed unchanged across the
`models/`/`gormstore/` split above. That shim was deliberately removed:
`UserManager`'s methods now spell these types as `models.Actor` etc.
directly, and every caller (this package's own tests included) imports
`github.com/aosanya/mwanachama-backend-actor/models` alongside the root
package rather than going through a re-export. The package doc comment
moved from `models.go` to `doc.go` since there's no longer an aliases file
to hang it on. Tradeoff accepted knowingly: callers are now coupled to this
repo's internal `models/` subpackage path, so a future reshuffle of that
package is a breaking change for callers — small cost, given the type set
is tiny (four types) and stable.
**`mwanachama-backend-shared`, `-git`, and `-taskmanager` were NOT touched**
and keep entitygraph as-is — this migration was scoped to this repo only.
**`mwanachama-backend-api-gateway`'s wiring (`cmd/server/stores.go`,
`cmd/server/user_instances.go`) now calls a `NewUserManager` signature and a
`DefaultUserSchema` that no longer exist and will not compile** until it's
updated in a follow-up change — not done here, by explicit scope decision.

## Porting notes

- `user.go`'s `UserManager` interface and `models/`'s domain types
  (`Actor`, `Group`, `Registration`) port the gateway's
  `internal/domain/member.Member`/`.Registration` and
  `internal/domain/chapter.Chapter` field-for-field, with three changes:
  `Member` → `Actor` (this repo's own naming-consistency pass — the
  gateway's own domain, routes, and Postgres tables keep the word "member"
  unchanged, and the `instance` string the gateway mounts this package under
  stays whatever the gateway chooses, e.g. `"member"`), `Chapter` → `Group`
  (DSN-1698 decision 3 — the gateway is responsible for presenting it under
  an org-configurable label; this repo never hardcodes the word "Chapter"
  anywhere an id belongs), and `Registration` becomes a real row in its own
  table (composite primary key `(actor_id, group_id)`), not a graph edge.
- `gormstore.DefaultTableNames(instance)` (wrapped by root `tables.go`)
  builds the three physical table names (`Actors`/`Groups`/`Registrations`)
  for one mounted copy of this package — `instance` names the gateway's
  mounted copy (e.g. `"member"`), preserving the multi-instance-mount
  capability the old `DefaultUserSchema(instance)` provided via
  `StorageCollection` labels. `gormstore.Migrate(db, tables)` runs GORM
  `AutoMigrate` scoped to those names. A known limitation, not fixed: GORM
  derives index/constraint names from the row struct, not the runtime table
  name, so migrating two *different* instances into the same Postgres schema
  could collide on constraint names — only relevant if/when a second
  instance is ever mounted (today only `"member"` is). Timestamps stay
  hand-formatted RFC 3339 strings (not GORM-managed `time.Time` columns) —
  see `models/time.go`'s `TimeLayout` doc for why plain `time.RFC3339Nano`
  isn't safe for sort order.
- **`Hierarchy` and `Level` moved here, 2026-09-07 (DSN-1699 gap 1,
  resolved the other way).** The gap was left open by the original design
  session and this repo's first build took the DEFAULT it named — hierarchy
  and level stayed behind in the gateway's own Postgres. That default rotted
  silently: the gateway's raw-SQL store kept querying `hierarchy`/
  `chapter_level` tables no migration created any more (they were dropped
  during an earlier cutover and nothing replaced them), producing a 500 on
  every `POST /v1/hierarchies`. Resolved by moving `Hierarchy`/`Level` in
  here instead, alongside `Group`/`RoleKind` — `models/hierarchy.go`,
  `gormstore/hierarchy.go`, `hierarchy_impl.go`. The deciding factor: every
  row that references a Level (`Group.LevelID`, `Group.AnchorLevelOverrideID`,
  `RoleKind.LevelID`) already lives in this package, so `DeleteLevel`'s two
  refusals — worn by a Group, scoped to by a RoleKind — can now both be
  checked in one native transaction. The RoleKind half of that check had
  been silently dropped from the gateway's old split-store version after
  role folded in here and nobody re-wired it; this restores it.
  `gormstore.Migrate`'s partial unique index on `(hierarchy_id) WHERE
  is_default_anchor` is the new database-level home for the old gateway
  migration 000033's `chapter_level_one_default_anchor` guarantee — still no
  FK from `Group.LevelID`/`.AnchorLevelOverrideID` to the new Level table,
  same posture as `ParentID`, for the same multi-instance-table-naming
  reason `gormstore/group.go`'s `GroupRow` doc explains. One behavior change
  from the retired gateway stores: hierarchy/level ids now mint as UUIDs
  (`HierarchyRow`/`LevelRow`'s `BeforeCreate`, matching `GroupRow`/
  `RoleKindRow`) rather than the old `hier-`/`lvl-` sequence-numbered
  strings — no client or test asserted on that format. Another: each
  mounted instance now gets its own hierarchy/level tables
  (`<instance>_hierarchies`/`<instance>_levels`) rather than one shared
  gateway-side store across every instance — no effect today (only
  `"member"` is mounted) but worth knowing if a second instance is ever
  added.
- **`Registration.IsHome` carries no exclusivity.** DSN-1698 decision 8
  drops "one home chapter per member" outright, not relocated — do not
  reintroduce a uniqueness check on `is_home`, in this repo or in a caller.
- **`MoveGroup`'s cycle/subtree check is a Go walk, not a recursive CTE.**
  The gateway's Postgres `chapter_store.go` used `WITH RECURSIVE`. Now that
  storage is a real relational table, a recursive CTE would work here too —
  but `group_impl.go` still lists every group and walks the parent chain in
  memory, unchanged from the entitygraph version, since rewriting it wasn't
  part of the storage-swap's scope. Fine at org scale; revisit if a
  deployment's group count ever makes this a real cost.
- **No act-log writing here, by design.** The gateway's
  `member.Repository.Deregister` composes and writes a
  `custody.ChapterActLogEntry` in the same transaction as the delete
  (DEV-1343/DEV-1344). This repo's `UserManager.Deregister` only removes the
  registration row and returns the prior `Registration` (which still
  carries `IsHome` — the one fact a caller needs to compose the same act
  after the fact). `custody` is a gateway-internal domain this repo must not
  depend on; the gateway adapter is where the act gets composed and written,
  in its own, separate call — not inside the same Postgres transaction as
  the row delete, since the two now live behind different interfaces even
  though they happen to share one physical database. This is a deliberate,
  noted loss of atomicity, not an oversight.
- **Member search (the console directory, DEV-1034) is NOT built here.**
  `member.Searcher.Search` composes member + chapter + role + verification
  data, and role/verification stay gateway-side. The gateway already has a
  working composer for exactly this shape —
  `internal/store/memory/member_search.go`'s `MemberSearchService`, which
  depends only on the gateway's own domain interfaces (`member.Repository`,
  `chapter.Repository`, `role.Repository`, `verification.Repository`) and
  is therefore reusable unchanged. Nothing in this repo needs to reproduce
  that composition.
- Straightforward CRUD files (`actor_impl.go`, `group_impl.go`,
  `registration_impl.go`) only ever call `m.db` (a `*gorm.DB`) directly for
  querying, going through `gormstore`'s row types and `*ToRow`/`*FromRow`
  converters for storage shape — `gormstore/` is the seam that keeps GORM
  out of `models/`'s and the root package's public API.
- **`Attributes` is now a validated, constrained map, not a free-form blob —
  2026-09-04.** `models/property.go`'s `Property` (name/label/range/
  required/unique) is this repo's own T-Box, one step past the gateway's
  `member_data_property` (which has no `required`/`unique`, only
  name/label/range/options). `models.DefaultActorProperties()` and
  `models.DefaultGroupProperties()` are the built-in catalogs; `CreateActor`
  and `CreateGroup` run `models.ValidateAttributes` (Required/Range/Options,
  pure Go) and the root package's `attributes.go`'s `checkUniqueAttributes`
  (Unique, a `datatypes.JSONQuery` lookup) before writing. `Phone` and
  `Email` are NOT Actor struct columns any more — they moved into
  `Attributes["phone"]`/`Attributes["email"]` as the catalog's two default
  entries, both `Unique`, neither `Required` (mirrors the gateway's
  nullable-phone reality — `registerDevice` mints a member before a number
  is given, and migration 000058 forbids one outright for agentic actors).
  `Group` gained an `Attributes` field it did not have before, validated
  against `DefaultGroupProperties` (empty today). The Go-level unique check
  is a pre-check only, not the sole guard: `gormstore.Migrate`'s
  `syncUniqueAttributeIndexes` creates a matching partial unique index
  (`attributes ->> 'name'` on postgres, `json_extract` on sqlite) so a
  concurrent write that races past the pre-check still fails at the
  database. If a future caller needs an `Attributes` *update* path (there
  is none yet — only Create validates), it must run the same two checks.

**`role` and `dashboard` folded in, 2026-09-06 (DEV-1659/DEV-1660/DEV-1661,
todo_actor_absorb.md).** The gateway's `internal/domain/role` (role kinds +
assignments) and `internal/domain/dashboard` (the roles-with-members view +
rollup counts) ported the same way `member`/`chapter` did: `models.RoleKind`/
`models.ActorRoleAssignment` (renamed `MemberID`→`ActorID`, `ChapterID`→
`GroupID`, mirroring `ActorGroupAssignment`'s own rename) and
`models.RoleWithMembers`/`models.GroupDashboard`, GORM rows in `gormstore/`,
new methods on the same `UserManager` interface rather than a second
interface — `CreateRoleKind`/`ListRoleKinds`/`GetRoleKind`/`RetireRoleKind`/
`UnretireRoleKind`/`GrantRole`/`GetRoleAssignment`/`RevokeRole`/
`StepDownRole`/`EndRoleOnEviction`/`EndRoleOnDeparture`/
`ListRoleAssignmentsForGroup`/`ListRoleAssignmentsForActor`/`GroupDashboard`.
`routes/role.go` adds the four plain shells (`CreateRoleKind`/`ListRoleKinds`/
`GetRoleKind`/`GetRoleAssignment`) to `Routes()`.

**No act-log writing here either, same reason as Deregister's.** DEV-1658
(landed in the gateway *before* this port, precisely so this port would not
carry the dependency) moved the gateway's `role`/`member` act-log
composition — `GrantAct`/`RevokeAct`/`StepDownAct`/`RetirementEvent`/
`EndOfMembershipAct` — into `internal/api/http`, the gateway's own layer.
`RetireRoleKind` still takes an `actorID` because `RetiredBy` is a genuine
domain field, not audit dressing; every other role method that used to take
an actor purely to write a custody row (`UnretireRoleKind`, `GrantRole`,
`RevokeRole`, `StepDownRole`) takes none now.

**`GroupDashboard`'s wire shape is not the gateway's**, same as `Group`'s:
`GroupID`/`actor_ids` where the gateway's `GET /v1/chapters/{chapterID}/
dashboard` says `chapter_id`/`member_ids`. Translating between the two is
the gateway's job at cutover (DEV-1662, not done here), the same job it
already does presenting `Group` as "chapter" everywhere a client sees one.

**`RetireRoleKind`'s transaction carries no explicit row lock.** The
gateway's Postgres store took one (`SELECT ... FOR UPDATE`) so a grant and a
retirement racing on the same kind couldn't both land in the state G229
refuses; no method in this repo takes an explicit lock anywhere else
(`AssignGroup`'s upsert has the identical shape of race), so this port
matches that existing risk posture rather than introducing the first
lock — a known gap, not a silent regression.

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
  `internal/store/entitygraph` (rename pending the gateway's own follow-up
  change), not here.

## MCP tools

A domain library that already ships its own `routes/` HTTP package (this
one included) *may* also ship its own `mcp/` package exposing the same
manager's methods as Model Context Protocol tools, once a hosting product
actually needs an AI agent to reach it. `mwanachama-backend-agency`'s
`mcp/` package (added 2026-09-10, AG13) is the reference implementation —
follow its shape rather than re-deriving one from scratch.

This is deliberately a *may*, not a *must*: MCP tool code has otherwise
always lived in a hosting gateway, not the domain library — see
`mwanachama-backend-api-shared`'s centralized `mcp_tools_taskmanager.go`/
`mcp_tools_git.go`, which exist there specifically because **two**
independent gateways (`api-gateway` and `api-kazi`) need the identical
tool set for those domains. That centralization pattern is still correct
for that problem and is not superseded by this one — don't migrate
taskmanager/git (or any other already-centralized domain) into their own
libraries just to match agency. Reach for an in-library `mcp/` package
only when a domain has exactly one primary hosting product, the way
agency currently has exactly one (`mwanachama-wakala-api`).

Two things make MCP a different shape than `routes/`, not just an MCP
version of it:

- **`ManagerResolver`, not a concrete manager.** `routes/`'s
  `Routes(m AgencyManager, names ResourceNames) []Route` gets away with a
  concrete manager because a mounting host rebuilds it fresh on every HTTP
  request, closing over whichever manager that request's own path segment
  resolves to. MCP's `mcp.AddTool` wires a handler onto the server *once*,
  at startup — there's no per-call rebuild point — so an `mcp/` package
  instead exports `type ManagerResolver func(ctx context.Context, id
  string) (Manager, error)` and a `RegisterTools(server, resolve)` that
  every tool closes over. The mounting host supplies `resolve`; a
  single-Agency host can ignore the id and always return the same manager,
  a multi-Agency host (like wakala-api's own Registry) resolves it per
  call. See `mwanachama-wakala-api/internal/api/http/mcp.go`'s
  `resolveAgencyManager` for the multi-instance case.
- **Every List/Match tool must return an object, never a bare slice.**
  MCP's spec defines `CallToolResult.structuredContent` as a JSON object;
  the go-sdk does not enforce this and will happily serialize a bare slice
  return type straight into it, which passes the go-sdk's own (lenient)
  test client but fails a spec-strict one with "expected record, received
  array". Every List/Match tool in `mwanachama-backend-agency/mcp` returns
  `ListResult[T]{Items: [...]}` for exactly this reason — reuse that
  exported type (or its equivalent) rather than returning `[]T` directly.

A library adopting this pattern should add one line to its own CLAUDE.md
next to its `routes/` entry: "`mcp/`, matching agency's `mcp/` package
exactly. See actor/CLAUDE.md." — the same propagation mechanism `routes/`
itself already uses.
