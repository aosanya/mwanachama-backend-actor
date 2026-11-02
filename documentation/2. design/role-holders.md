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
under `actor.role_assignment.create`, reads the kind first and then the actor's
live seats of that kind at that group.

- An unknown kind is `ErrRoleKindNotFound` (404). A retired kind is
  `ErrKindRetired` (409).
- An actor who already holds a live seat of the same kind at the same group is
  `ErrSeatAlreadyHeld` (409), so a second grant of the same seat leaves one seat. An
  ended seat does not count: the person can be granted the kind again, and the
  history keeps both rows.
- `granted_by` is the caller, resolved from the session; a value in the body is
  ignored.
- The actor and the group are not read. A grant naming a person or a group
  that does not exist succeeds and writes the seat. What the Admin sees when
  she picks a name that is not a member is hub question Q121.
- The kind's `level_id` is not compared with the group's level (see "Not here"
  in [group-leaders.md](group-leaders.md)).
