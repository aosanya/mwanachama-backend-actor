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
