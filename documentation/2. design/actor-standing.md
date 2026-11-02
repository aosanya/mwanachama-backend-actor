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
  `ended_reason`. `definition_role_id`, `granted_by`, `granted_by_name`,
  `ended_at` and `ended_reason` are left out of the body when empty; the last
  two are empty while the seat is held. The role fields come from the seat's
  role kind, retired or not.

Both lists are `[]` on the wire for a person with nothing, never `null`.

- An unknown or soft-deleted actor is `ErrActorNotFound` (404), unlike the
  registrations and seats reads, which answer `[]`.
- A granter that is no actor (a host's operator id, or a seat the first run
  created with no granter) leaves `granted_by_name` out and the seat listed.

## Seats that point at nothing

A seat is listed whether or not the group and the kind it names still exist.
`group_name` and `role_name` are always on the wire, so a ghost shows as `""`.
`TestStandingKeepsLiveSeatsOfDeletedAndMissingGroupsAndUnknownKinds` pins all
three cases:

- **A live seat in a soft-deleted group** is still shown as held: no `ended_at`,
  the deleted group's `group_name`, and `leads_group: true` when its kind leads a
  group. Three reads disagree about that group. `seats` shows it held and
  led, `registrations` drops it, and `ListLedGroups` (`GET /groups/led`) leaves
  it out.
- **A seat on a group id with no row** keeps its `group_id` and has an empty
  `group_name`.
- **A seat on a kind id with no row** keeps its `kind_id` and has an empty
  `role_name`, `leads_group` and `staff` false, and no `definition_role_id`.

Seats at a group with no row exist in databases from before `1d30cc0`, when
`GrantRole` took any group id. Seats in a soft-deleted group exist there too, and
arise whenever a group is soft-deleted after its seat was granted: nothing in
this module ends the seats of a deleted group. `GrantRole` reads the kind first
and this module has no call that deletes one, so a seat on an unknown kind is a
row written outside the module. The read does not filter or flag any of the
three.

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
