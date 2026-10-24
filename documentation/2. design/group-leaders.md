# A group's leaders (ACT15, 2026-10-20)

Member app decision 71: **a group's leaders are the holders of a role kind
marked as leading the group, set by the agency.** Only leaders decide a
group's join requests (ACT13, [joining-groups.md](joining-groups.md)) and
issue its invitations (ACT14, [invitations.md](invitations.md)). wakala-api
WK112 is the consumer.

## The mark

`role_kind` gained one declared field, `leads_group` (bool, off by default).
`CreateRoleKind` takes it with the rest of the kind, and
`SetRoleKindLeadsGroup(ctx, kindID, leads)` changes it afterwards, because a
kind has no other edit path and the kinds that already exist need marking. A
missing kind is `ErrRoleKindNotFound`.

`Provision` adds the column to an existing `role_kind` table, through the same
`columnsDeclaredSince` list ACT13 introduced (shared's S52), so every kind that
existed before reads back as not leading. Nothing leads until the agency says
so.

## Who is a leader

An actor leads a group when they hold an **active** seat
(`actor_role_assignment.active`) **in that group** of a kind whose
`leads_group` is set. So:

- A seat leads only the group it is in. Leading one ward says nothing about
  the ward beside it, or the groups above or below it.
- A seat that ended (revoked, resigned, evicted, departed, role removed)
  leads nothing.
- Clearing the mark on a kind ends leadership for every holder at once;
  setting it makes every current holder a leader at once. Nothing is copied
  onto the seat.
- A retired kind leads nothing (decision 412, ACT71). `leadingSeats` reads
  only kinds with `retired_at` empty, so a role the definition drops stops
  leading at once. Hand retiring still refuses a kind with live seats, so in
  practice only a definition sync retires a kind under held seats, and that
  sync ends them too.
- Registration at the group is not checked. `GrantRole` has never required
  it, and the seat is the fact decision 71 names.

`IsGroupLeader(ctx, groupID, actorID)` answers that question and is exported
so a host can ask it directly (to admit a leader at its own gate, say).
`ListLedGroups(ctx, actorID)` reads every group, not deleted, that the actor
leads, oldest first.

## The refusal

`ApproveJoinRequest`, `DeclineJoinRequest` and `InviteToGroup` refuse a
caller who does not lead the group with `ErrNotGroupLeader`, declared **403**.

- Deciding: the request is read first (a missing one is still
  `ErrJoinRequestNotFound`), then leadership of the request's group is checked
  inside the same transaction as the conditional write, so a refused decision
  writes nothing and the request stays pending.
- Inviting: the group is read first (a missing one is still
  `ErrGroupNotFound`), then leadership, **before** the invited actor is looked
  up. A non-leader therefore cannot learn whether an actor id exists, or is
  already in the group, by trying to invite it.

The decider and the issuer are the ones the route table already binds
`from: caller` (`decided_by`, `invited_by`), so no body can name a leader.

**There is no bypass.** The module has no notion of an operator or an agency
owner, and no document says one decides a group's join requests in place of
its leaders, so none is built. A host that wants its operator to act on a
group has to give that operator a leading seat there; whether it should is an
open question for the owner, recorded on WK112.

## Route table

| Method | Path | Action | Bound from the session |
|---|---|---|---|
| GET | `/groups/led` | `actor.group.list_led` | `actor_id` |
| PUT | `/role-kinds/{kind_id}/leads-group` | `actor.role_kind.set_leads_group` | |

## Not here

- A read of a group's leaders. Nothing asked for one, and
  `ListRoleAssignmentsForGroup` with the kinds already answers it.
- Scoping a leading kind to a level; `level_id` already limits where any kind
  may be granted.
