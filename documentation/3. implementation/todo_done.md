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

## ACT18 — invite by member code — 2026-10-21

Owner, member app decisions 132, 136. ddce255 (merged as 0328590):
`invite_to_group` takes the invitee as exactly one of `actor_id` or
`actor_code` (neither or both is `ErrInvalidInvitation`, 400);
`InviteToGroup(ctx, groupID, actorID, actorCode, invitedBy, expiresAt)`. A
code is trimmed and upper-cased, matches only this agency's live actors, and
an unknown one is the same `ErrActorNotFound` 404 as an unknown id; the leader
check still runs first. New `get_my_actor` (`GET /actors/mine`,
`actor.actor.read_mine`, tool `actor_get_my_actor`) returns the caller's own
record with its `code`; no actor row is a 404. `invitation_code_test.go`, a
wire test; `invitations.md` "Inviting by member code (ACT18)". wakala-api
c68c4f6 passes the code through; self-test `resident-invitations` 51/51.
Verified: `go build`, `go vet`, `gofmt`, `go test ./...` ok. Granting
`actor.actor.read_mine` to residents is WK133.

## ACT19 — a reason on a join-request decline — 2026-10-21

Owner, member app decision 133. b0eeaa8 (merged as f678960): new field
`group_join_request.decline_reason` (json `decline_reason`), returned on every
read including the applicant's own; `decline_join_request` takes an optional
body `reason`, trimmed, at most 500 characters (`ErrDeclineReasonTooLong`);
a reason with `silent: true` is `ErrQuietDeclineReason` (400) and writes
nothing; an appeal, a re-ask after approval and an open group's policy
settling a request clear it. `Provision` adds the column to an existing table.
`join_decline_reason_test.go`, wire tests; `joining-groups.md` "A reason on a
decline (ACT19)". wakala-api b0949cd passes it through; self-test
`group-leaders` 60/60. Verified: `go build`, `go vet`, `gofmt`,
`go test ./...` ok. Carrying the reason into the Inbox notice is MF66's.

## ACT20 — find and join a group by its code — 2026-10-21

Owner, member app decision 137. cf00803 (merged as 69a50b0, a call site
adapted to ACT19's signature): `FindGroupByCode` answers `group` (hidden
included, with `code`), `join_mode`, the caller's `state`
(`member`/`requested`/`declined`/`none`) and the request when there is one;
`JoinGroupByCode` follows the join mode exactly as `JoinGroup` does, a hidden
group included. A code is trimmed and upper-cased; empty, unknown, deleted and
another agency's code all answer `ErrGroupNotFound`. New operations
`GET /groups/by-code?code=` (`actor.group.find_by_code`) and
`POST /groups/by-code/join` (`actor.group.join_by_code`), caller-bound; listed
joinable groups carry `code`. `join_by_code_test.go`,
`routes/join_by_code_test.go`; `joining-groups.md` "Finding a group by its
code (ACT20)", hidden rule amended in `invitations.md`. Declared table 68
routes and tools with ACT18. Verified: `go build`, `go vet`, `gofmt`,
`go test ./...` ok; wakala-api builds and tests with only board-pinned reds.
Mounting it is WK134.

## ACT21 — random member and group codes — 2026-10-21

Owner, member app decisions 179–181. 1917879 (merged as ee621e2): actors and
groups gain `public_code`, `M-`/`G-` and N hyphen-joined groups of three from
an alphabet with no look-alikes, drawn with crypto/rand at creation, unique per
agency behind a partial unique index; a collision redraws with one more group
(eight in a row is an error). N comes from a new `code_format` role (default 3,
1–12 allowed, a bound the agent chose and documented in `public-codes.md`),
read on every draw, through `GET`/`PUT /code-format`
(`actor.code_format.read`/`.set`, MCP tools too; 70 routes). `Provision`
backfills every actor and group, deleted rows included. Invite-by-code (ACT18)
and find/join-by-code (ACT20) match `public_code` only; `AC-<n>`/`G-<n>` answer
as unknown. `invitations.md`, `joining-groups.md` updated;
`models/schema_generated_test.go` hand-edited for Group (developer's
`structure.json` still needs `public_code`). wakala-api c8dcbbf moves the
`resident-invitations` self-test onto the new code, 9bbeada one join-by-code
test. Verified: `go build`, `go vet`, `go test ./...` ok here; wakala-api
`go test ./...` only board-pinned reds, `make self-test` only the known
failures. Mounting the setting and owner-gating it is WK143.

## ACT22 — a join request keeps every decision — 2026-10-21

Owner, member app decision 184. 61952b0 (merged as f9be67e): a new declared
`join_decision` object (request, group, actor, state, silent, decided_by,
decided_at, reason), indexed on request and actor. `decide()` inserts one
through `recordDecision` in the same transaction as the request update, and
never overwrites one; the request still carries its latest. The applicant
reads their own told decisions through `GET /join-decisions/{decision_id}`
(`actor.join_decision.read_mine`, MCP tool `actor_get_my_join_decision`, the
actor bound from the caller); a quiet one, another actor's and an unknown id
all answer the same 404 (`ErrJoinDecisionNotFound`). `ListJoinDecisions` is a
manager method only (71 routes). An open group's auto-approval records
nothing. `joining-groups.md` updated. Left for WK144: no decision id reaches
the applicant over HTTP yet. Verified: `go build`, `go vet`, `go test ./...`
ok here; wakala-api builds and vets, `go test ./...` only `_OpenHole_` reds.

## ACT23 — a member's priority for each of their groups — 2026-10-21

Owner, member app decision 192. ea18fed (merged as 8b32806): `registration` gains
an enum `priority` (`very_high`, `high`, `normal`, `low`, `very_low`,
default `normal`), added to existing tables by `Provision`; `AssignGroup`
writes `normal` on a new registration and keeps it on a re-assign, and older
rows read back as `normal`. `PUT /groups/{group_id}/priority`
(`actor.group.set_my_priority`, MCP tool `actor_set_my_group_priority`) binds
the actor from the caller; a group the caller is not in answers 404, an unknown
value 400 (`ErrInvalidPriority`). `ListMyGroups` returns `[]MyGroup`, the
group plus `priority` (72 routes). `group-priority.md` added. Mounting it is
WK147. Verified: `go build`, `go vet`, `go test ./...` ok here; wakala-api
builds and vets, `go test ./...` only `_OpenHole_`/`_OpenBug_` reds.

## ACT24 — a member's own Forms week-split choice — 2026-10-21

Owner, member app decision 196. e41bce5 (merged as e83379d): the `actor` object
gains `forms_week_split` (bool, default `true`), added to existing tables by
`Provision`, so older rows read back `true`; `CreateActor`/`EnsureActor` write
`true`. `GET`/`PUT /actors/mine/forms-week-split`
(`actor.actor.read_my_forms_week_split`, `actor.actor.set_my_forms_week_split`;
MCP tools `actor_get_my_forms_week_split`, `actor_set_my_forms_week_split`)
bind the actor from the caller and answer `{actor_id, forms_week_split}`; a
caller with no actor record reads `true` and a set answers 404 (74 routes).
`forms-week-split.md` added beside `group-priority.md`. Mounting it is WK151
(week split), which must grant both actions to residents. Verified: `go build`,
`go vet`, `go test ./...` ok here; wakala-api builds and vets against it.

## ACT25 — removing a member from a group or every group, history kept — 2026-10-21

262cf4d. A declared `group_removal` record (actor, group, removed by, removed
at). `RemoveFromGroup` deletes the registration (so every group and
caller-bound read answers as for a stranger), ends the actor's active roles
there as an eviction and records the removal; `RemoveFromAllGroups` does it
for every group; `ListRemovals` reads the history. The actor row is never
touched, so past answers and messages keep their author; rejoining is an
ordinary assignment. `removing-members.md`. Left for wakala-api WK158: the
agency standing (instance membership) and declaring the operations' routes
and actions. Verified: `go build`, `go vet`, `go test ./...` ok; wakala-api
builds against it.

## ACT27 — the field worker role, listed with who holds it — 2026-10-21

4ef60d2 (merged as 8c7d5b5). Granting and revoking a role kind per member
already existed; added `ListRoleHolders(ctx, kindID, activeOnly)` and
`GET /role-kinds/{kind_id}/holders` (`actor.role_kind.list_holders`, tool
`actor_list_role_holders`): every seat of one kind across groups with the
holder's display name, ended seats kept in the history. `role-holders.md`.
The field worker kind is an ordinary role kind; seeding it is wakala-api
WK169. Verified: `go build`, `go vet`, `go test ./...` ok; wakala-api builds.

## ACT28 — an agency's operators decide join requests — 2026-10-21

3ffa8d9 (merged as 65db236). `OperatorApproveJoinRequest` and
`OperatorDeclineJoinRequest` decide like the leader methods (decided once,
history, quiet decline reasonless) without `mustLead`; declared as
`actor.join_request.operator_approve`/`operator_decline` at
`POST /join-requests/{request_id}/operator-approve`/`operator-decline`,
`decided_by` from the caller. The leader path is unchanged. Left for
wakala-api WK157: grant both actions only to the Instance's owners and
agents. Whether leaders keep deciding stays open (WK157).
`joining-groups.md`. Verified: `go build`, `go vet`, `go test ./...` ok
(route and tool counts now 77 with ACT27); wakala-api builds.

ACT29 · Group names in several languages (decision 256) · 2026-10-22.
6fd057f (merged as 9c2cbab). New `group_name_translation` object, unique on
(group, language); `SetGroupNameTranslation` (a changed name clears review),
`ReviewGroupNameTranslation` (reviewer from the caller),
`ListGroupNameTranslations`, `GetGroupInLanguage` and `GroupNamesInLanguage`
falling back to the original; four declared addresses
(`actor.group_name.list`/`translate`/`review`, `actor.group.read_in_language`).
`group-name-translations.md`. Open with the owner: limiting tags to the
agency's languages is left to wakala-api; a group has no original-language
field. Verified: `go build`, `go vet`, `gofmt`, `go test ./...` ok (route and
tool counts 81); wakala-api builds.

## ACT30 — a role kind remembers the definition role it came from — 2026-10-22

Member app decision 269. 311a533 (merged as 776adf9): `role_kind.definition_role_id` in the blueprint and `models.RoleKind` (added to `columnsDeclaredSince`), `RoleKindForDefinitionRole`, and `SyncDefinitionRoles` (renames linked kinds, retires kinds whose role left the definition, leaves seats and hand-made kinds alone, idempotent); `definition-roles.md`. Open with the owner: should active seats under a dropped role end, and with what reason? If a dropped role comes back, revive the old kind or make a new one? Verified: `go build`, `go vet`, `go test ./...` ok; wakala-api builds.
