# Roster paging

HS303, from the hub's decision 98 (owner, 2026-10-29, Q84): a roster of people
in the app shows the first 50 with a Show more button, paged by the person's
key, and a search box finds a person beyond the first 50. Performance review
entry 15 found the cost this removes: the roster listing returned every
registration of a group, and its outsider filter read the whole seats table
on every call.

## The page

`ListGroupRoster(ctx, groupID, afterActorID, limit)` returns the registrations
of one group ordered by `actor_id`, the actor half of the registration's
composite key `(actor_id, group_id)`. Within one group the actor id is unique,
so it is a complete cursor: the next page is asked for with `afterActorID` set
to the last `actor_id` received. No table and no index was added; the page
reads the `group` index on `actor_group_assignment`, then takes the first rows
past the cursor.

- `limit` of zero or less answers 50 (`DefaultRosterPageSize`); anything above
  200 (`MaxRosterPageSize`) answers 200, so a caller cannot ask for the whole
  roster in one call.
- A page shorter than the limit is the last one. A page of exactly the limit
  may or may not be, so the app asks once more and hides Show more when the
  answer is empty.
- Outsiders (`outsiders.md`) are left out. A page is filled from the following
  rows when some of those it read were outsiders, so a short page always means
  the end of the roster, never an outsider in the way.

## The outsider filter

`outsidersAmong` answers which of a given list of actor ids are outsiders:
every active seat they hold is an outsider seat (ACT74). Both seat reads in it
(the outsider seats and the ordinary seats that clear the actor) are narrowed
to those ids, in chunks of 500, and use the `actor` index on
`actor_role_assignment`. The cost of a page is the page, not the number of seats
in the agency.

`ListActorsForGroup` keeps its contract (the whole roster, oldest first,
outsiders left out) for server-side callers that need all of it: the
dashboard's member count, survey announcements, notification previews. It now
reads the group's registrations and asks `outsidersAmong` about those actors
only. `HomeCounts` still uses the unscoped `outsiderActors` subquery, because
it counts across every group.

## The route

`GET /groups/{group_id}/assignments` (`actor.assignment.list_for_group`,
tool `actor_list_actors_for_group`) calls `ListGroupRoster` and takes two
optional query parameters, `after` and `limit`. The body is still the array of
registrations, so a caller that sent neither parameter now receives the first
50, in key order rather than oldest-first.

## Search

Not built here. Finding a person beyond the first 50 is a name search over the
group's actors, which the roster listing does not do: it pages by key only.
The frontend half of HS303 needs a search that answers by name; this repo has
none for a group's roster yet.

## Tests

`roster_page_test.go` and `routes/roster_page_test.go`: three pages of a
120-member group by key, the cap, a page that stays full across outsiders, an
unknown group, and statements captured from the database showing that every
seat read of a page and of the whole-roster listing is narrowed by
`actor_id IN`.
