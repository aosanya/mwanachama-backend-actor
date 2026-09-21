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

## ACT1 — a raced duplicate phone is a `ErrDuplicateAttribute`, not `ErrDuplicateID` — 2026-09-21

`classifyDuplicateID` (`actor_impl.go`) now tells the violated index apart: a Postgres `23505` whose `ConstraintName`, or a sqlite `UNIQUE constraint failed` message, names one of `gormstore.syncUniqueAttributeIndexes`'s `<table>_attr_<name>_uniq` indexes returns `ErrDuplicateAttribute` (400); any other unique violation stays `ErrDuplicateID`. `routes/actor.go`'s `actorStatusFor` also gained an `ErrDuplicateID` → `409` arm, since a genuine caller-supplied-id collision previously fell through to an opaque 500 as well — a small addition beyond the row's literal text, in the same function. The Postgres branch is written from pgconn's API but only the sqlite path is exercised by the tests. The two pins became `TestCreateActor_RacedDuplicatePhoneIsDuplicateAttribute` and `TestActorRoutes_RacedDuplicatePhoneReturns400`, still using the forced-interleave barrier. Not done, as the row itself noted: `group_impl.go`'s `CreateGroup` has no equivalent classification (unreachable while no Group property is `Unique`), and `NextCode`'s unlocked read-then-write race. Verified: `go build ./... && go vet ./... && go test ./...` and `go test -tags=integration ./...` green (the raced tests also under `-race`); `gofmt -l .` reports only `models/role.go`, untouched and already unformatted.

## ACT2 — renaming a GroupType no longer defeats `DeleteGroupType`'s guard — 2026-09-21

Took the rename-cascade option from the row: `EditGroupType` (`grouptype_impl.go`) now updates the type and, when the name changed, sets `node_type` to the new name on every Group in the type's hierarchy that carried the old one, in one transaction. Groups follow their type, so the guard, which reads the current name, keeps seeing them. **This changes documented behaviour**: `EditGroupType` used to write only Name/Singular/Plural/LevelID and leave `Group.NodeType` alone as free text; its doc comment now says a rename carries the hierarchy's Groups across. The cascade is scoped to the type's own hierarchy, but `DeleteGroupType`'s count still spans all hierarchies, as before. The alternatives (guard by id, or refuse a rename while worn) were not taken. The pin became `TestDeleteGroupType_StillRefusedAfterRename`: the Group's `NodeType` becomes the new name, the delete is refused, and the type still exists. Verified: `go build ./... && go vet ./... && go test ./...` and `go test -tags=integration ./...` green (the raced tests also under `-race`); `gofmt -l .` reports only `models/role.go`, untouched and already unformatted.
