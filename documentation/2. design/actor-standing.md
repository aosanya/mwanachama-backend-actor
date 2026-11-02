# One read for a person's standing (HS690, 2026-11-02)

A manager looking at one person's page used to make five reads: the actor, her
registrations, her seats, the role kinds and the groups. `role_kind.list` sits
only in `role_kind.manage`, too broad to hand a manager, and the seats and
registrations reads answer `200 []` for an actor that does not exist.
`GetActorStanding(actorID)` answers all of it in one call.

## The read

`GET /actors/{actor_id}/standing`, action `actor.actor.standing`, MCP tool
`actor_get_actor_standing`. The body is an `ActorStanding`:

- `actor_id` and `display_name`.
- `registrations`, oldest first: `group_id`, `group_name`, `since` (the
  registration's `created_at`, which `AssignGroup` keeps across a re-assign).
  A deleted group is left out, as in [my-groups.md](my-groups.md); a hidden
  group is kept.
- `seats`, oldest grant first, ended seats included: `id`, `group_id`,
  `group_name`, `kind_id`, `role_name`, `definition_role_id`, `leads_group`,
  `staff`, `granted_at`, `granted_by` (the granter's actor id as stored),
  `granted_by_name` (the granter's display name), `ended_at` and
  `ended_reason`. The last two are absent while the seat is held. The role
  fields come from the seat's role kind, retired or not.

Both lists are `[]` on the wire for a person with nothing, never `null`.

- An unknown or soft-deleted actor is `ErrActorNotFound` (404), unlike the
  registrations and seats reads, which answer `[]`.
- A granter that is no actor (a host's operator id, or a seat the first run
  created with no granter) leaves `granted_by_name` empty and the seat listed.
- A seat in a deleted group keeps its `group_name`; only registrations drop
  deleted groups.

## Who may read it

Nothing in this module decides that. The action joins `member.view` in the
agency's permissions catalogue, beside `actor.actor.read` and
`actor.role_assignment.list_for_actor`, so a caller holding `member.view` may
read any person's standing and a caller without it is refused with 403 by the
mount's `Authorize`. Pinned by `TestStandingIsGatedOnItsOwnAction`.

## Route table

| Method | Path | Action | Bound from the session |
|---|---|---|---|
| GET | `/actors/{actor_id}/standing` | `actor.actor.standing` | |
