# Hub copy diff

`mwanachama-backend-actor` is the custodian of the actor module. The hub (`projects/hub-steward/code/wakala-api/modules/actor`) holds a copy made on 2026-11-04 (hub commit a7d64d10) and changed it afterwards. Every committed change made in that copy exists here (hub issue ACT73, part of ISS-105). The backend is ahead of the hub copy by the MUN678 files until the hub copy carries them: `name_check.go`, `actor_impl.go`, `group_impl.go`, `hierarchy_impl.go` and `routes/routes.go` (backend commit 17cf2b1). Until then the script prints a difference for each of those five files.

## Command

```
scripts/hub-copy-diff.sh [path-to-hub-modules-actor]
```

The default path is `../projects/hub-steward/code/wakala-api/modules/actor`, relative to this repository's parent folder. The module path is read from `go.mod`. The script and `scripts/hubdiff` are the ones of `mwanachama-backend-shared` (S65), pointed at one module. To compare against the committed hub only, pass a `git archive` export of the hub's `modules/actor` folder.

## What it does

1. Copies every `*.go` and `*.json` file of the hub folder and of this repository into two temporary folders, leaving out `cmd`, `documentation`, `scripts` and `.git`.
2. Rewrites the hub's imports: `wakalaserver/modules/actor` becomes `github.com/aosanya/mwanachama-backend-actor` and `wakalaserver/modules/shared/` becomes `github.com/aosanya/mwanachama-backend-shared/`.
3. Runs every Go file on both sides through `scripts/hubdiff`, which parses it, drops all comments and prints it with `gofmt`, so comments and layout do not show as differences.
4. Runs `diff -r backend hub`.

## Result of 2026-11-16

Against the committed hub (dev at 169958ee), run on a `git archive` export of this repository at 17cf2b1 and of the hub's `modules/actor` folder, the script prints a `diff` for five Go files and 79 `Only in backend` lines, and nothing `Only in hub`. The three JSON files and every other non-test Go file are identical after the rewrite.

The five `diff` files are the MUN678 name rules (see `name-rules.md`), which the hub copy does not have yet:

| File | Difference |
|---|---|
| `name_check.go` | the backend adds `maxNameCharacters`, a required check and a length check on top of the plaintext check, and `checkActorDisplayName`, `checkActorPersonNames`, `checkGroupName` and `checkHierarchyName`; the hub file holds `checkName` and `checkActorNames` with the plaintext check only |
| `actor_impl.go` | the backend calls `checkActorDisplayName` and `checkActorPersonNames`; the hub calls `checkActorNames` |
| `group_impl.go` | `CreateGroup` and `EditGroup` call `checkGroupName` in the backend; the hub checks for an empty name and calls `plaintext.CheckLine` |
| `hierarchy_impl.go` | `CreateHierarchy` and `RenameHierarchy` call `checkHierarchyName` in the backend; the hub calls `checkName` |
| `routes/routes.go` | the hub keeps `maxGroupName` and the length check in the route wrapper of the group edit; the backend removed both because the manager enforces the limit |

The 79 `Only in backend` lines are all test files:

| Difference | Reason |
|---|---|
| Only here: the tests of the root package, `models` and `routes` that are not hub files, among them `testdb_test.go`, `postgres_integration_test.go`, `removal_test.go`, `name_check_test.go`, `display_name_free_test.go`, `routes/routes_test.go` and, for MUN678, `name_length_test.go` and `routes/name_length_test.go` | The hub copy of 2026-11-04 was made without them, or the hub commit carried no actor test and the test was written here before the code (red first). They test code that is identical in both, except the two MUN678 files, which test the five files above. |
| `routes/routes_test.go` counts 99 routes and tools | It lists the route table; it moved with the routes the hub added. |
| `leader_test.go` and `outsider_test.go` reuse one role kind of a name | A role kind name is unique since hub ebf32240, so the helpers that seated several leaders or outsiders no longer create a second kind of the same name. |
| `models/schema_generated_test.go` gained `Boundary` and `NameKey` | It is generated-style and has no hub counterpart; it was updated by hand with the two columns. |

Against the hub working tree the diff also shows `delegation_impl.go`. That file holds an edit in the hub that is not committed (a delegation `ends_at` that is not RFC 3339 or not in the future is refused with a reason). It is not carried over until the hub commits it.

## The hub commits carried over

One commit here per hub commit, each naming the hub hash, in hub order:

| Hub | Here | What |
|---|---|---|
| dc47ce41 | 38cb383 | a seat carries an approval limit |
| 91df5c3e | 1c50082 | a group carries a boundary |
| 82b8d095 | 4a0c05a | set a seat's approval limit, AuthoriseApproval |
| 36cebd57 | 994d7d0 | groups at a point |
| d0ddefd7 | 05d4250 | operations for set-limit and authorise (MUN13) |
| 31c46340 | b10c321 | a missing required query argument answers 400 |
| 374aa5c8, 0ec4e98b, 404f2854 | 9bba506, 077b2ff, d381132 | cover tests: delegations, group name translations and populations, hierarchies and groups |
| b5862cb8, 0f2aaa01, 587563ef, ff75f884, b712f0ad | 2b5aeb1, 0d84cfc, 1d8de67, b10df25, 1b1c394 | cover tests: join requests, invitations, actors, seats and standing, routes |
| 9f93f73e | 142631c | removing a member ends pending join requests (MUN86) |
| 0e40790a | 64e261f | next member number claimed by one atomic upsert (MUN106) |
| ccb82777 | 6527bbe | removing a member ends every role seat (MUN95) |
| 27df7d13 | 48e6961 | a group name with a control character answers 400 (MUN101) |
| 1bea6f05 | 676928a | a resident who leaves loses seats, roster places and pending requests (MUN97) |
| 25042349 and 3845cb09 | 19c19d6 | two groups of one agency cannot share a name (one commit: the first alone does not build) |
| 2f23368f | b9b6a48 | group names fold to NFC |
| 0b5c0fe8 | 7a52158 | group names fold with NFKC and drop trailing dots and spaces |
| 7b0b14d6 | 54bc25c | the fold lives in `plaintext.FoldDrawnName` in shared |
| da99df0a | ac67914 | names that draw nothing are refused |
| 5762d51f | 8eb8be1 | `invited_by` omitted when empty |
| ebf32240 | 017742b | a role kind or hierarchy whose name is taken is refused with 409 (MUN316) |
| c702fd78 | 1645100 | a look-alike display name is refused with 409 (MUN210) |

The hub commit a7d64d10 is the copy itself and has nothing to carry.
