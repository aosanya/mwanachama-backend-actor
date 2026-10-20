# A member reads their own groups (ACT17, 2026-10-20)

Member app decision 99: **a member reads the groups they belong to, with each
group's name.** wakala-api WK120 is the consumer, and opens the read to
residents; the member app's MF7 shows it.

## The read

`ListMyGroups(ctx, actorID)` returns every group the actor is registered in,
as the full `Group` (so its `name` comes with it), **oldest registration
first** — ordered by the registration's `created_at`, which `AssignGroup`
keeps across a re-assign, not by when the group was made. Ties fall back to
the group id.

- **Registration is the only fact read.** An open join, an approved request,
  an accepted invitation and an operator's `AssignGroup` all end in the same
  registration row, so all of them are covered — including the operator's,
  which leaves no join request behind. A pending, declined or appealed
  request is not a registration and does not appear.
- A deleted group is left out, as in `ListLedGroups`. A hidden group is kept:
  hiding takes a group out of the joinable list, not out of its members'.
- An actor with no registrations, and an empty or blank actor id, get an
  empty list (`[]` on the wire), never an error.

## Bound to the caller

The operation is `list_my_groups`, declared beside `list_led_groups` and
shaped like it: its one argument, `actor_id`, is bound `from: caller`. A
query string or a body naming another actor is ignored, and the path carries
no actor at all. Pinned over a real mux with a real caller by
`TestOverTheWireACallerReadsOnlyTheirOwnGroups`.

`GET /groups/mine` sits beside `GET /groups/{group_id}`; the mux prefers the
literal segment, the same way `/groups/led` and `/groups/joinable` already do,
and the test reads both addresses on one server.

## Route table

| Method | Path | Action | Bound from the session |
|---|---|---|---|
| GET | `/groups/mine` | `actor.group.list_mine` | `actor_id` |

The MCP tool is `actor_list_my_groups`.

## Not changed

`list_groups_for_actor` (`GET /actors/{actor_id}/assignments`) stays as it
is: path-bound, registration rows with group ids only, for operators.
