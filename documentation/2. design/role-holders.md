# Who holds a role kind

ACT27, member app decision 240. The field worker role is an ordinary role
kind: an owner creates it, grants it to a member with `GrantRole` and
revokes it with `RevokeRole`. Nothing in this module names it; what the
role lets a holder do (collecting answers for somebody else) is an action
another module declares and `mwanachama-backend-permissions` grants
(PM20, forms F36), and the host seeds the kind (wakala-api WK169).

What was missing was the list an owner reads: who holds this kind.
`ListRoleHolders(kindID, activeOnly)`, served at
`GET /role-kinds/{kind_id}/holders` under the action
`actor.role_kind.list_holders`, answers every seat of one kind in any group,
oldest grant first, each with its holder's display name.

- An unknown kind is `ErrRoleKindNotFound` (404), not an empty list.
- `active_only=false` keeps ended seats with their `ended_reason`, so a
  revocation stays in the history.
- A seat whose actor row is gone still lists, with an empty
  `display_name`, rather than failing the whole list.

## What granting a seat checks (HS625, HS689)

`GrantRole` (`role_assignment_impl.go`), served at `POST /role-assignments`
under `actor.role_assignment.create`, runs in one transaction and reads, in this
order, the kind, the actor, the group and the actor's live seats of that kind at
that group. The first read that refuses ends the grant and nothing is written.

- An unknown kind is `ErrRoleKindNotFound` (404). A retired kind is
  `ErrKindRetired` (409).
- An actor who already holds a live seat of the same kind at the same group is
  `ErrSeatAlreadyHeld` (409), so a second grant of the same seat leaves one seat. An
  ended seat does not count: the person can be granted the kind again, and the
  history keeps both rows.
- The same rule is held by the database: `actor_role_assignment` declares the
  unique index `held_once` on (`actor_id`, `group_id`, `kind_id`) where `active`,
  so two grants that both pass the check cannot both land. The one that loses
  gets the same `ErrSeatAlreadyHeld`. `Provision` creates the index with the
  table, so a database built from nothing has it. A database whose tables
  already exist does not: see "An older database" below.
- `granted_by` is the caller, resolved from the session; a value in the body is
  ignored.
- An actor or a group that does not exist, or is deleted, is
  `ErrActorNotFound` or `ErrGroupNotFound` (404) and nothing is written. A
  seat that already exists is not touched when its actor row is later deleted:
  `ListRoleHolders` still lists it with an empty `display_name`. What the Admin
  sees when she picks a name that is not a member is hub question Q121.
- The kind's `level_id` is not compared with the group's level (see "Not here"
  in [group-leaders.md](group-leaders.md)).

### An older database

`Provision` (`provision.go`) returns at once on a database that `upToDate`
calls up to date. `upToDate` checks that every declared table exists, that no
legacy table name is left, that the columns in `columnsDeclaredSince` are there
and that no coded row lacks its code or public code. It checks no index. So the
`held_once` index reaches only a database `Provision` builds from nothing:

- On a database whose tables already exist and that is otherwise up to date,
  `Provision` changes nothing and `held_once` is never created. Two grants that
  race past the Go check can both land there, and the second `GrantRole`
  does not answer `ErrSeatAlreadyHeld`.
- The other side. When a later schema change makes `upToDate` false on a
  database that already holds two live seats of one actor, group and kind,
  `Provision` goes on to `spec.Migrate`, which runs
  `create unique index if not exists … (actor_id, group_id, kind_id) where active = true`,
  and the database refuses it (on SQLite, `UNIQUE constraint failed`).
  `Provision` returns that error, and no object after `actor_role_assignment`
  in the blueprint is migrated. How long it keeps failing depends on what made
  `upToDate` false. It fails on every call while that is still pending: a row
  without a code (the codes are filled after the migration, which never gets
  that far), or a table that comes after `actor_role_assignment` in the
  blueprint. It fails
  once when `Provision` had already dealt with it before the index: a column
  added from `columnsDeclaredSince`, or a table that comes before
  `actor_role_assignment`. The next call then finds the database up to date,
  returns nil, and the index is still absent.
- In wakala-api, `buildActorManager` in `internal/agencies/registry.go` calls
  `Provision` before it builds the manager and returns its error, so while
  `Provision` fails, `ActorManager` fails for that agency and its actor routes
  have no manager to run on, until the duplicates are removed by hand.

The seat table's name is a hash of the instance, the mount and the object, so
it cannot be written here. It is `Spec.TableFor` of the object that fills the
role `actor_role_assignment`, and `go run ./cmd/ddl <spec.json> postgres`
prints it in the `…_held_once_idx` statement. The live seats that share an
actor, a group and a kind are found with:

```sql
select actor_id, group_id, kind_id, count(*) as seats
from <seat table>
where active = true
group by actor_id, group_id, kind_id
having count(*) > 1;
```

`Provision` does not look for duplicates before it creates the index, and no
routine in this module or in wakala-api runs the query. `cmd/seatduplicates`
runs it when someone runs the command, and only then. Which seat of each pair
stays, how the others end, and whether an older database gets the index at all
are the owner's decision (HS702); the command only reports.

### Finding the duplicates (`cmd/seatduplicates`)

```sh
POSTGRES_URL=<url> go run ./cmd/seatduplicates -instance <instance> [-mount <mount>] [-schema <schema>]
SQLITE_PATH=<file> go run ./cmd/seatduplicates -dialect sqlite -instance <instance> [-mount <mount>]
```

The command finds the seat table the way `cmd/provision` does, from
`SpecForMount(instance, mount)`; the group and role kind tables it joins for
names come from the same spec. An empty `-mount` is the default mount. In
wakala-api `buildActorManager` calls
`SpecForMount(inst.TableSlug, MountDirectory)`, so the instance is the agency's
`table_slug` (a column of `agency_instances`) and the mount is `directory`.
Every agency's tables sit in the registry's one database, told apart by the
instance prefix, so there is one run per agency.
`-schema` qualifies the three table names for a database whose tables are not
on the connection's search path; it takes a plain identifier and nothing else.

The database address is read from `POSTGRES_URL` (or `SQLITE_PATH` with
`-dialect sqlite`) and from no argument, so it reaches neither the shell history
nor the process list. A stray argument is refused. The command prints neither
the address nor any part of it: the address, its user and its password are
struck out of every error message before it is printed.

It never writes. On Postgres the query runs in a `READ ONLY` transaction; on
SQLite the file is opened with `mode=ro`, so the file and its directory can be
read-only on disk and no journal file is made. It runs the group-by above over
the live seats, joined back to the seats for their ids and to the group and the
role kind for their names. It does not create the index, does not call
`Provision`, and ends no seat.

One line per duplicated triple, then a count:

```text
actor=<id> group=<id> group_name="<name>" kind=<id> kind_name="<name>" seats=2 ids=<older>,<newer> oldest=<granted_at> newest=<granted_at>
duplicates: 1
```

- `ids` are the live seats of that triple, oldest `granted_at` first;
  `oldest` and `newest` are the least and greatest `granted_at` among them, and
  an empty one shows as `-`.
- An empty `group_name` or `kind_name` means that row is gone from its table;
  the triple is still listed.
- Every line is a triple on which creating `held_once` would be refused. An
  ended seat beside a live one, and the same actor holding the same kind in two
  groups, are not duplicates and are not listed.
- Exit 0: no live triple is shared, so the index could be created on this
  database. The index is still absent unless something created it; the command
  does not check for it. Exit 1: at least one is shared. Exit 2: it could not
  look: a missing variable, a connection that fails, an instance, mount or
  schema that holds no seat table, or a wrong flag. Nothing is printed on
  standard output for an exit 2.

What to do with the duplicates stays with the owner (HS702). The tests run the
command against SQLite files; the Postgres path, the `READ ONLY` transaction
and the connection, was not run against a Postgres server when this was
written.
