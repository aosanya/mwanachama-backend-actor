# A member's priority for each of their groups

ACT23, from the member app's decision 192 (owner, 2026-10-21).

## What is stored

A registration carries a `priority`: `very_high`, `high`, `normal`, `low` or
`very_low`, declared as an enum on the blueprint's `registration` object with
`normal` as its default. It is the member's own ranking of the group among
theirs, so it lives on the registration row, the one fact that is about this
actor in this group, and it is kept per member per instance, which is what
lets it follow the member to any device.

## Every registration starts at normal

`AssignGroup` writes `normal` on a new registration, and every way into a
group goes through it: an operator placing an actor, an open join, an
approved join request, an accepted invitation. A re-assign of a pair that
already exists keeps the priority the member chose. A row written before the
column existed reads back as `normal`; `Provision` adds the column to an
existing table through `columnsDeclaredSince`.

## Only the member sets it, and only their own

`PUT /groups/{group_id}/priority` (`actor.group.set_my_priority`,
`SetMyGroupPriority`) binds the actor `from: caller`, so a body naming another
actor is ignored and nobody can set somebody else's priority. A group the
caller is not registered in, a missing group and a call with no caller all
answer `ErrGroupNotFound` (404), the same refusal, so the call cannot be used
to learn which groups exist. A value outside the five is `ErrInvalidPriority`
(400).

## The groups-of-mine read returns it

`ListMyGroups` returns `MyGroup`: the group with the caller's `priority`
beside it, so the JSON is the group's own fields plus `priority`. The order
is unchanged, oldest registration first; ordering by priority is the
client's, as decision 192 sets it per screen. `ListGroupsForActor` and
`ListActorsForGroup` carry the priority on each registration too.

`Group` itself gained no field: it is the stored group, the same for every
member, and its shape is pinned by the generated schema test.
