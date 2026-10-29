# Outsider roles

ACT69, from the member app's decision 296 (owner, 2026-10-22): an outsider is
a member holding a narrow outsider role — a law firm's client, a school's
guardian, a tender's bidder — who sees only what is linked to them. One
sign-in and one app; outsiders are left out of member counts and groups.

## What is stored

`role_kind.outsider` is a bool, false by default, declared in
`actor.blueprint.json` beside `leads_group` and carried by
`RoleKind.Outsider`. It is set when the kind is created
(`CreateRoleKind`). `Provision` adds the column to a table created before it
(`columnsDeclaredSince`).

There is no address to flip the mark on an existing kind yet. Adding one would
add a method to `UserManager`, which every consumer compiles against, so it
is left for the row that needs it.

## Who is an outsider

ACT74, from decision 494 (owner, 2026-10-25): an actor is an outsider while
**every live seat it holds is an outsider seat** — at least one active seat
of an outsider kind and no active seat of any other kind
(`outsider_impl.go`'s `outsiderActors`). A member who also holds an outsider
seat stays a member: an ordinary seat beside it keeps them listed, counted
and able to join. When their last ordinary seat ends (revoke, step down)
and an outsider seat is left, they are left out; ending the outsider seat
brings them back with no further act.

An outsider seat never ends a group registration. Granting one writes only
the seat; the registration row stays, and the lists below hide it at read
time while the actor is an outsider.

This replaces ACT69's first reading, under which any active outsider seat
made the actor an outsider.

## What it changes

- `ListActorsForGroup` and `HomeCounts` leave outsiders out, so
  `GroupDashboard.MemberCount` does too. The registration row is untouched:
  the filter is a read-time lookup of which of the listed actors are
  outsiders (`outsidersAmong`), narrowed to those actors; `roster-paging.md`.
- `JoinGroup` (and joining by code, which shares `join`), `InviteToGroup`
  and `AcceptInvitation` refuse an outsider with `ErrOutsiderRole`, mapped to **409** in
  `actor.operations.json`. The refusal comes after the actor is found, so an
  unknown actor still answers as one. Accepting is refused too, so an
  invitation issued before the outsider seat cannot slip them in; declining
  still works.

## Not covered

`GroupDashboard`'s per-kind holder lists still name an outsider under their
own outsider kind, because that list is of seats, not members. `ListMyGroups`
and `ListGroupsForActor` are the actor's own view and are unchanged.
