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
