# Role kinds published from the agency definition

Member app decision 269 (2026-10-22): the agency definition's roles are the
role kinds people hold. Board row ACT30; wakala-api WK181 is the caller.

## The link

`role_kind.definition_role_id` holds the id of the definition role a kind was
published from. A kind made by hand leaves it empty and is never touched by a
definition sync. The id, not the name, is the link, so a rename in the
definition renames the kind and every seat held under it reads the new name.
A kind is created with the id set through the ordinary `CreateRoleKind`.

## The manager

- `RoleKindForDefinitionRole(ctx, definitionRoleID)` answers the kind
  published from one definition role, `ErrRoleKindNotFound` when none was,
  and `ErrDefinitionRoleIDRequired` for an empty id.
- `SyncDefinitionRoles(ctx, roles, actorID)` takes the definition's whole
  current list of roles, in one transaction:
  - a linked kind whose role is still listed under a different, non-empty
    name is renamed;
  - a linked kind whose role is still listed takes the role's `staff` tick;
  - a linked kind whose role is no longer listed is retired, `retired_by`
    the actor publishing, and its live seats end.
  It answers which kinds were renamed and which retired. A kind already
  retired is not retired again, so a repeated sync changes nothing.

## Dropping a role ends its seats

Member app decision 412 (2026-10-24, ACT71). Unlike `RetireRoleKind`, which
still refuses a kind with live seats (`ErrKindHasLiveAssignments`), a sync
retires a dropped kind whatever is held under it and, in the same
transaction, ends every active seat of it with `ended_reason` `role_removed`
and `ended_at` the retirement time. The seat rows stay, so the history of who
held the role is kept. A retired kind leads no group (see
[group-leaders.md](group-leaders.md)).

## Staff kinds

Member app decision 411 (ACT71). `role_kind.staff` (bool, off by default)
marks a kind the definition ticks as a staff role, a Street inspector or a
Clerk. `DefinitionRole.Staff` carries the tick into the sync, which writes it
onto the kind whenever it differs, and `CreateRoleKind` accepts it on a new
kind. Whether a person is staff (owners, agents, or a holder of a staff kind)
is the host's answer; this module only stores the tick. Capabilities are not
copied from the definition's permissions, which the permissions module holds per
kind.

## Not done here

- Creating kinds for definition roles that have none yet stays with the
  caller (`CreateRoleKind` with `DefinitionRoleID` set), as does deciding
  their other fields.
- A role that returns to the definition does not unretire its old kind;
  reviving it is the caller's call to `UnretireRoleKind` (decision 412). The
  seats the drop ended stay ended.
- No route or MCP tool exposes either method: publishing the definition is
  wakala-api's act, and it calls the manager.

## Storage

The columns `definition_role_id` and `staff` are added to an existing table on provision through
`columnsDeclaredSince`, the same path `leads_group` took, and `definition_role_id` is indexed as
`definition_role`.
