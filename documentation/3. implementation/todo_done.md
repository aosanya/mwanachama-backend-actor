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

## ACT13 — a group can be joined: join policy, hidden flag and join request — 2026-10-20

The repo is already a declared domain (blueprint, operations, specstore; ACT5's
shape landed, though this board and CLAUDE.md still describe `gormstore/`), so
the change is declared. `Group` gains `join_mode` (`open`/`confirmed`,
`CreateGroup` fills `confirmed` and refuses anything else) and `hidden`, set
through the new `SetGroupJoinMode` and `SetGroupHidden`; `EditGroup` is
unchanged. A new `group_join_request` role holds group, actor, state
(`pending`/`approved`/`declined`), `appeal`, `message` (≤ 500), `silent`,
`requested_at`, `decided_by`, `decided_at`, unique on (group, actor).
`JoinGroup` registers at once on an open group, writes or updates the one
request on a confirmed group, turns a declined request into an appeal keeping
its first `requested_at`, and answers `ErrGroupNotFound` for a hidden group.
`ApproveJoinRequest` writes the registration in the same transaction; a
decided request is `ErrJoinRequestDecided` (409). `DeclineJoinRequest` stores
`silent`, so the host raises CM33's decline event only when it is false.
`ListJoinableGroups` returns groups not deleted, not hidden and not
registered. Nine declared operations (47 → 56), actor and decider bound from
the caller. `Provision` adds the two group columns to an existing table;
existing groups read back `confirmed` and not hidden. `joining-groups.md`;
`join_test.go` and a route test through a real mux. Commits e00a3ed (a gofmt
fix to `models/role.go` already failing on main) and e454a48. `go test ./...`
green in the main checkout, gofmt clean; wakala-api builds against it, red
only on its deliberate `_OpenHole_` tests. For WK99 and the owner: nothing
says which role kinds make a group leader, so the decide operations check
nothing and the host must gate them; whether a quietly declined member keeps
seeing Requested is a product call; `hidden` sits beside the older
`discoverable`.

## ACT14 — an invitation to a hidden group — 2026-10-20

Owner, member app decision 56. 7d8ecab adds a declared `group_invitation`
object, at most one per group and actor. `InviteToGroup` records the issuer
from the caller and, like ACT13's decider, checks nothing about whether
they lead the group. `AcceptInvitation` writes the registration in the same
transaction and settles any pending join request; `DeclineInvitation` records
the decline and registers nothing; only the invited actor may answer, anyone
else gets 404. `expires_at` is optional and caller-supplied (no document
names a duration, so none was invented); a lapsed invitation is refused
(409) and left out of the open list. Inviting again renews the row; inviting
an actor already in the group is `ErrAlreadyInGroup` (409). Six declared
operations (56 → 62), issuer and answerer bound from the caller.
`invitations.md`; `invitation_test.go` and a route test through a real mux.
`go build`, `go vet`, `go test ./...` green in the main checkout, gofmt
clean; only new `UserManager` methods, so wakala-api builds against it. For
WK99: the six actions need gating, and who counts as a leader is still the
owner's question.

## ACT15 — a role kind can lead its group — 2026-10-20

Owner, member app decision 71. efb1d5d gives `role_kind` a declared
`leads_group` (off by default; `Provision` adds the column to an existing
table, so every existing kind reads as not leading), set on `CreateRoleKind`
or later through `SetRoleKindLeadsGroup` (`PUT /role-kinds/{kind_id}/leads-group`,
`actor.role_kind.set_leads_group`). An actor leads a group while holding an
active seat of a leading kind in that group, and only there. New
`IsGroupLeader` for a host's own gate and a declared `ListLedGroups`
(`GET /groups/led`, `actor.group.list_led`, caller-bound). `ApproveJoinRequest`,
`DeclineJoinRequest` and `InviteToGroup` refuse a non-leader with
`ErrNotGroupLeader` (403) after the missing-row 404; a refused decision writes
nothing, and a refused invite never reveals whether the invited actor exists.
No operator bypass (WK112's note: refuse anyone else). `group-leaders.md`.
`go build`, `go vet`, `go test ./...` green, gofmt clean. wakala-api 93cd95e
seats the owner as Chair in its two decision tests and has `resident-groups`
check the owner's refusal; admitting leaders at its gate is WK112.

## ACT16 — an invitation's lifetime is a system setting — 2026-10-20

Owner, member app decision 81. 4123c8c gives `NewUserManager` options
(`NewUserManager(db, s, opts ...Option)`, existing callers unchanged):
`WithInvitationLifetime(d)`, `DefaultInvitationLifetime` 14 days, a
non-positive lifetime refused at construction. `InviteToGroup` without
`expires_at` stores the issue time plus the lifetime; an issuer's own date is
kept, and a renewal without a date restarts it. Invitations issued before
ACT16 with no date still never lapse (no backfill). `invitations.md` gains
"How long an invitation lasts (ACT16)"; `invitation_lifetime_test.go`.
`go build`, `go vet`, `go test ./...` green, gofmt clean. wakala-api b7515fe
reads `INVITATION_LIFETIME` (`14d` or a Go duration; unset is 14 days; invalid
stops the boot) and hands it to every agency's directory; self-test
`invitation-lifetime` 12/12.

## ACT17 — a member reads their own groups — 2026-10-20

Owner, member app decision 99. Landed as 47ab4c3: `list_my_groups`
(`GET /groups/mine`, action `actor.group.list_mine`, MCP tool
`actor_list_my_groups`) over `ListMyGroups(ctx, actorID)`. The actor comes
from the session only; it returns every group the caller is registered in,
name included, oldest registration first, covering open joins, approved
requests, accepted invitations and operator registrations; pending requests
and deleted groups are left out, hidden groups stay. `list_groups_for_actor`
is unchanged. Design page: `documentation/2. design/my-groups.md`. Verified:
`go build ./... && go vet ./... && go test ./...` ok (root, models, routes),
`gofmt -l` clean; wakala-api builds and tests against it with only its three
board-pinned reds (WK65, WK80, WK84). Residents are admitted by wakala-api WK120.

