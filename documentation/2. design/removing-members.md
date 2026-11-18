# Removing a member (ACT25, 2026-10-21)

Member app decision 220 (owner, 2026-10-21): an operator ends a membership
in one group, or every membership an actor holds, and the history is kept.
wakala-api WK158 is the consumer.

## What a removal does

- `RemoveFromGroup(actor, group, removedBy)` deletes the registration row
  and writes a `group_removal` record: the actor, the group, who removed
  them (`removed_by`, required, the mount's caller) and when (`removed_at`,
  minted by the store). An actor not registered at the group is
  `ErrNotInGroup`; an empty remover is `ErrRemoverRequired` and removes
  nothing.
- `RemoveFromAllGroups(actor, removedBy)` does the same for every group the
  actor is registered at and returns one record per group.
- `ListRemovals(actor)` reads an actor's removals, oldest first.

## Why the registration row goes

The registration row is what every group read and every caller-bound read
already asks, so deleting it is what makes a removed actor answer exactly as
a stranger everywhere, with no read having to learn a new filter. The
removal is the history, kept in its own object and never rewritten, the same
way `join_decision` keeps decisions apart from the request they decide.

The actor record is never touched, so past answers and messages keep their
author. Rejoining is an ordinary fresh registration — through an invitation
or a request — and leaves the earlier removals in place.

## Not in this module

The actor's standing in the agency (wakala-api's instance membership) is the
host's, not this module's: `RemoveFromAllGroups` ends every group
membership, and the host ends the agency standing beside it. The operation
is not yet declared in `actor.operations.json`; the host calls the manager.
Every active role the actor holds in the group ends with it, as an
eviction, so a removed leader leads nothing there.

## No group removal is declared (ACT76, 2026-10-25)

Member app decision 459: group removal is not declared in the people module.
`actor.operations.json` used to carry `deregister`
(`DELETE /actors/{actor_id}/assignments/{group_id}`, action
`actor.assignment.delete`, tool `actor_deregister`), which deleted the
registration row with none of decision 431's rules and wrote no
`group_removal` record. Only wakala-api's mount kept it closed. The operation
is gone, so neither the route table nor the tool list offers it, and a host
that forgets to close it cannot expose it.

`Deregister` stays on `UserManager`: `RemoveFromGroup` calls it to delete the
registration row before writing the removal record. A host removes a member
through its own route over `RemoveFromGroup`, as wakala-api's WK158 does.

## One registration, declared again over RemoveFromGroup (SW609, 2026-11-18)

A committee page that lists its members from `actor_list_actors_for_group`
needs a tool that takes one person off that list, and revoking the seat leaves
them registered. `actor.operations.json` declares `unassign_group` again:
`DELETE /actors/{actor_id}/assignments/{group_id}`, action
`actor.assignment.delete`, tool `actor_unassign_group`. It differs from the
operation decision 459 removed in what it calls. It is `RemoveFromGroup`, not
`Deregister`, so the registration row goes, every active seat the person holds
in that group ends as an eviction, and a `group_removal` is recorded against
the caller, who is never taken from the body. A person not registered in the
group is 404 (`ErrNotInGroup`); a call with no caller is 400
(`ErrRemoverRequired`) and removes nothing. The person's other groups and
their actor record are untouched.

Who may call it is the mount's permission on the action, as for
`actor_assign_group`. The wakala-api rules of WK158 (an agent removes
residents only, the last owner stays) belong to its own route and are not part
of this tool.
