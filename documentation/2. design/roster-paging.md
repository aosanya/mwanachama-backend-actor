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

HS318. `ListGroupRoster` takes a last argument, `search`, and the route a query
parameter `q`. The text is trimmed and, when it is not empty, the page keeps
only the registrations whose actor's display name, first name or last name
contains it, whatever the case. The match is a `LIKE` over the group's actors
with `%`, `_` and `\` escaped, so they read as plain text. It narrows the same
`group_id` read the page already does, so the cursor works unchanged: a
search finds a person beyond the first 50 and its own pages are asked for with
`after`, 50 at a time, and a short page still means the end. The outsider
filter applies to the matches as it does to the roster.

## The group's leaders

HS318. `ListGroupLeaders(ctx, groupID)` answers who leads one group as a list
of `{actor_id, display_name}` and nothing else: the actors holding an active
seat in that group of a role kind that leads its group
(`group-leaders.md`), ordered by actor id. It is served at
`GET /groups/{group_id}/leaders` under `actor.group.list_leaders`, tool
`actor_list_group_leaders`. A group with no leaders, or an unknown group,
answers an empty list. A screen that says who an application waits on reads
the name here.

## Tests

`roster_page_test.go` and `routes/roster_page_test.go`: three pages of a
120-member group by key, the cap, a page that stays full across outsiders, an
unknown group, and statements captured from the database showing that every
seat read of a page and of the whole-roster listing is narrowed by
`actor_id IN`.

`roster_search_test.go`, `group_leaders_list_test.go` and the last test of
`routes/roster_page_test.go`: the name search with its case and its
wildcards, a person beyond the first page found and the matches paged, the
search kept inside its group, and the leaders named only for an active
leading seat in that group.
