# A group's leaders (ACT15, 2026-10-20)

Member app decision 71: **a group's leaders are the holders of a role kind
marked as leading the group, set by the agency.** Only leaders decide a
group's join requests (ACT13, [joining-groups.md](joining-groups.md)), apart
from the host-vouched operator path below (ACT28), and only leaders issue its
invitations (ACT14, [invitations.md](invitations.md)). wakala-api WK112 is the
consumer.

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
leads, oldest first. `ListGroupLeaders(ctx, groupID)` (`leader_impl.go`, HS318)
reads who leads one group, served at `GET /groups/{group_id}/leaders`; it is
described in [roster-paging.md](roster-paging.md).

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

**On the leader path there is no bypass.** The module cannot tell an operator
or an agency owner from a member, so `ApproveJoinRequest`, `DeclineJoinRequest`
and `InviteToGroup` ask every caller for a leading seat. Deciding a join
request without one is a separate pair of methods, `OperatorApproveJoinRequest`
and `OperatorDeclineJoinRequest` (ACT28, `join_impl.go`,
[joining-groups.md](joining-groups.md)), which the host vouches for; inviting
has no such pair. A host that wants its operator to invite has to give that
operator a leading seat there; whether it should is an open question for the
owner, recorded on WK112.

## Route table

| Method | Path | Action | Bound from the session |
|---|---|---|---|
| GET | `/groups/led` | `actor.group.list_led` | `actor_id` |
| GET | `/groups/{group_id}/leaders` | `actor.group.list_leaders` | |
| PUT | `/role-kinds/{kind_id}/leads-group` | `actor.role_kind.set_leads_group` | |

## Not here

- Scoping a leading kind to a level. A kind's `level_id` is stored but not
  enforced: `GrantRole` (`role_assignment_impl.go`) never compares it with the
  group's level, and only `DeleteLevel` reads it, to refuse deleting a level a
  kind names.

## Group roles and group actions (hub decisions 121, 128, 130)

Today a role kind marked `leads_group` is the only thing that gives a person a
leader's actions in a group: deciding its join requests and inviting to it. The
hub owner has decided that groups carry group roles (Admin, Vice Admin,
Coordinator) and that group actions, such as deciding a join request or
removing a member, are assigned to those group roles in the agency design, the
Admin being the one who decides who joins and leaves. This repo does not do
that yet: it has no group role and no assignment of actions to one, and
`RemoveFromGroup` asks no leading seat of its caller. The hub board rows HS612
and HS613 carry it.
