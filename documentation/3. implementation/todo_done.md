# mwanachama-backend-actor — completed work

## Initial build — 2026-09-03

Extracted `mwanachama-backend-api-gateway`'s `member` and `chapter` domains
onto the `mwanachama-backend-taskmanager` pattern, per DSN-1698/DSN-1699 (see
the gateway's `documentation/2. design/todo.md`). Built: `schema.go`
(`DefaultUserSchema`), `models.go`, `user.go` (`UserManager` interface),
`member_impl.go`, `group_impl.go`, `registration_impl.go`, `converters.go`,
`errors.go`, and the unit + Postgres-integration test suite. Wired into the
gateway via new adapters in `internal/store/entitygraph` (both backends),
migration `000061_member_tables`, and a not-yet-run backfill migration —
see the gateway board rows for the full account and real test output.

## Instance-mounting parameterization — 2026-09-03

`DefaultUserSchema()` → `DefaultUserSchema(instance string)`, so the gateway
can mount this package more than once (each with its own Postgres table
prefix and its own HTTP route prefix); the two `TypeDefinition.StorageCollection`
labels (`Member`/`Group`) now derive from `instance` instead of being
hardcoded to `"member_"`. `schema.Schema.ID`/`.Tag` stay fixed across
instances. Updated the one internal call site (`postgres_integration_test.go`,
now `DefaultUserSchema("useri")`). See the gateway's
`documentation/2. design/todo.md` row DSN-1700 for the full account,
including the gateway-side `UserInstance` declaration list and the route
move this made possible.

## Member → Actor rename — 2026-09-04

Renamed the `Member` type and every derived identifier to `Actor`
throughout this repo — `UserManager`'s `CreateMember`/`GetMember`/
`GetMembers`/`SetMemberDisplayName`/`ListMembers`/`ListGroupsForMember`/
`ListMembersForGroup` become `CreateActor`/`GetActor`/`GetActors`/
`SetActorDisplayName`/`ListActors`/`ListGroupsForActor`/
`ListActorsForGroup`; `Registration.MemberID` → `.ActorID`
(`json:"actor_id"`); `RelHasMember` → `RelHasActor` (`"has_actor"`);
`ErrMemberNotFound`/`ErrInvalidMember` → `ErrActorNotFound`/
`ErrInvalidActor`; the schema's `TypeDefinition.Name`/`DisplayName` "Member"
→ "Actor"; `member_impl.go` → `actor_impl.go`, `member_test.go` →
`actor_test.go`. Scoped to this repo only — the gateway's own `member`
domain, HTTP routes, and Postgres tables (and the `instance` string it
mounts this package under, e.g. `"member"`) are unaffected and keep the
word "member"; the gateway's existing `internal/store/entitygraph` adapters
that call into this package's API will need matching updates on the
gateway side to keep compiling.
