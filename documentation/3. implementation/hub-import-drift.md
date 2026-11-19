# Hub import drift

`mwanachama-backend-actor` is the owner of the actor module. The hub (`projects/hub-steward/code/wakala-api/modules/actor`) holds a copy and is to import this repository instead of keeping the copy. Before it can, this repository has to hold everything the copy holds. This page lists every difference found on 2026-11-19 and what was done with it. `hub-copy-diff.md` describes the earlier state and the commits carried over before this one.

## How it was measured

`scripts/hub-copy-diff.sh` was run against the hub working tree (no uncommitted change in `modules/actor`, hub `dev`). The script rewrites the hub's imports `wakalaserver/modules/actor` and `wakalaserver/modules/shared/` to this repository's and to `mwanachama-backend-shared`, drops comments, formats both sides with `gofmt` and runs `diff -r`. After the rewrite 29 files differ, 8 files exist only in the hub and 81 exist only here (tests).

## Only in the hub

| File or behaviour | What it is | Disposition |
|---|---|---|
| `approval_outcome_test.go`, `ApprovalOutcome` and `ApprovalOutcomeFor` in `approval_limit.go` | The answer to an approval of an amount: allowed, allowed above the limit, or refused with the largest limit held | Moved |
| `approves_above_limit` on a role kind: `actor.blueprint.json`, `models/role.go`, `models/definition_role.go`, `definition_role_impl.go`, `provision.go` (`columnsDeclaredSince`) | A setting on each seat kind that lets its holder approve above the limit written to the seat, the approval being flagged for the council. Off by default | Moved. The definition-role sync copies the setting to the role kind |
| `attribute_properties_impl.go`, `attribute_properties_test.go`, `AttributeProperties` in `user.go`, operation `list_attribute_properties` (`actor_list_attribute_properties`) in `actor.operations.json` | Lists the attribute names a person's record accepts, with label, range, unique and required | Moved. The operation count rises from 95 to 96 |
| `RangeDate`, `NotFuture`, `checkDay` in `models/property.go`, the `date_of_birth` property in `models/actor.go`, `date_attribute_test.go` | A date attribute written `YYYY-MM-DD`, optionally refused when it has not come yet, and the built-in Date of birth property | Moved |
| `ErrDuplicateGroupTypeName`, `checkGroupTypeNameFree` in `kind_hierarchy_name.go`, the call in `grouptype_impl.go`, `grouptype_name_unique_test.go`, the 409 entry in `actor.operations.json` and the sentinel in `routes/routes.go` | Two group types of one hierarchy cannot share a name (folded the way group names are) | Moved |
| `group_version.go`, `ErrStaleVersion`, `Group.Version` in `models/group.go`, the call in `EditGroup`, `GroupEdit.version` in `routes/tools.go`, the 409 entry in `actor.operations.json`, the sentinel in `routes/routes.go`, `routes/group_version_test.go` | A group edit that sends the `last_updated` it read is refused when the group changed since, through `versioncheck` in shared | Moved. Merged by hand into this repository's `EditGroup`, which keeps its own name rules |
| `removal_atomic_test.go` | Tests that `RemoveFromGroup` ends the registration, the seats and the removal record in one transaction | Moved. The code is already here |

## Only in this repository

| File or behaviour | Disposition |
|---|---|
| The name rules in `name_check.go` (required, trimmed, at most 200 characters, folded) and their callers in `actor_impl.go`, `group_impl.go`, `hierarchy_impl.go`; `routes/routes.go` no longer repeats the length check of a group edit | Kept. The hub copy has the plaintext check only. See `name-rules.md` |
| 81 test files, among them `testdb_test.go`, `postgres_integration_test.go`, `name_length_test.go`, `routes/routes_test.go` | Kept. They test code that is the same on both sides, or the name rules above |
| `cmd/`, `scripts/`, `documentation/` | Kept. The hub copy leaves them out by design |

## Differing beyond the import path

| File | Difference | Disposition |
|---|---|---|
| `delegation_impl.go` | The hub refuses a delegation whose `ends_at` is not RFC 3339, naming `ends_at` in the reason, and refuses one whose `ends_at` is not in the future. Here a malformed `ends_at` is refused without the reason and a past `ends_at` is accepted and the delegation is already ended | Reason wording moved. The refusal of a past `ends_at` is not moved and is an owner question, because the two behaviours differ for the same operation (see below) |
| `delegation_cover_test.go` | Follows `delegation_impl.go`: the hub test expects the refusal, this one creates a delegation that has already ended and checks it does not act | Kept as is, until the owner answers |
| `group_members_test.go`, `routes/assign_group_tool_test.go` | Helper names and invented person and group names only. The hub test file defines helpers that this repository defines in other test files | Kept. No behaviour differs |

## Hub wiring that is not part of the module

Not moved here. The hub's HTTP layer and permission catalogue stay in the hub and keep working against the module's exported names.

| Where in the hub | What it does with the module |
|---|---|
| `internal/api/http/step_approval_refusal.go`, `internal/agencies/approval_limits.go` | Call `ApprovalOutcomeFor` to flag or refuse an approval above the limit |
| `internal/api/http/definition_role_kinds.go`, `modules/agency/**` | Carry `ApprovesAboveLimit` from an agency's role into the definition role the module syncs |
| `internal/api/http/resident_actor.go` | Lists `actor.actor.list_attribute_properties` among the actions a resident may call |
| `modules/agency/models/permissions_catalog.json` | The permission catalogue: names the actor actions each permission code carries |
| `internal/api/http/*.go` (actor routes, tool filters) | Mount the module's routes and tools and filter the tool list per caller |

## Owner question

A delegation created with an `ends_at` in the past: should it be refused, as the hub does, or accepted and already ended, as this repository does? Nothing was changed for it. Until it is answered, the hub import must not remove the hub's refusal without the owner choosing.

## Result of the merge

After the merge `scripts/hub-copy-diff.sh` prints no `Only in hub` line. Ten files still differ, each for a reason in the tables above: the name rules (`name_check.go`, `actor_impl.go`, `group_impl.go`, `hierarchy_impl.go`, `routes/routes.go`), the delegation refusal of a past `ends_at` (`delegation_impl.go`, `delegation_cover_test.go`), test helper and invented names (`group_members_test.go`, `routes/assign_group_tool_test.go`) and the order of two functions in `kind_hierarchy_name.go`. `go test ./...` passes on in-memory SQLite.

Pinned counts that changed because the union requires it:

| Where | From | To | Why |
|---|---|---|---|
| `routes/routes_test.go`, routes and tools of the declared table | 101 | 102 | The operation `list_attribute_properties` adds one route and one tool |
| `models/schema_generated_test.go`, fields of `GroupEdit` | 3 | 4 | `GroupEdit.Version` for the stale-edit check |
| `CLAUDE.md`, operations declared in `actor.operations.json` | 95 | 102 | The text was already behind the file (101); it now matches |
