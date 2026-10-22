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
  - a linked kind whose role is no longer listed is retired, `retired_by`
    the actor publishing.
  It answers which kinds were renamed and which retired. A kind already
  retired is not retired again, so a repeated sync changes nothing.

## Retiring keeps who held it

Unlike `RetireRoleKind`, which refuses a kind with live seats
(`ErrKindHasLiveAssignments`), a sync retires the kind whatever is held under
it and leaves every seat row exactly as it was, active or ended. A retired
kind can no longer be granted, which is what dropping a role from the
definition means; whether the seats still held should then end is not
decided and is not done here.

## Not done here

- Creating kinds for definition roles that have none yet stays with the
  caller (`CreateRoleKind` with `DefinitionRoleID` set), as does deciding
  their other fields.
- A role that returns to the definition does not unretire its old kind;
  `UnretireRoleKind` is the caller's to use.
- No route or MCP tool exposes either method: publishing the definition is
  wakala-api's act, and it calls the manager.

## Storage

The column is added to an existing table on provision through
`columnsDeclaredSince`, the same path `leads_group` took, and is indexed as
`definition_role`.
