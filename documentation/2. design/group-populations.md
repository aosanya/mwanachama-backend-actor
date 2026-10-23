# Group populations

ACT34, 2026-10-23. Source: the municipality template's `council_actor.json`
(AGX14, member app decision 283), whose coverage wish needs "a population
figure per group, dated and sourced".

## Why

A group has members: the people who joined. Coverage is members over the
people the group serves, and that denominator is an estimate made outside the
module (a census, a register, a planner's count). Until now the module had
nowhere to hold it.

## Shape

`group_population` is its own declared object, not a field on `group`:

| Field | Meaning |
| --- | --- |
| `group_id` | the group the figure is for |
| `population` | a count, zero or more |
| `as_of` | the date the figure describes, `YYYY-MM-DD` |
| `source` | where it comes from, in the recorder's words; required |
| `recorded_by` | the caller, bound `from: caller`, never the body |
| `created_at` | when it was recorded |

A separate object because a figure is dated: a newer estimate is added beside
the old one, never written over it, so the history survives.

## Operations

| Address | Call | Action |
| --- | --- | --- |
| `POST /groups/{group_id}/population` | `RecordGroupPopulation` | `actor.group_population.record` |
| `GET /groups/{group_id}/population` | `LatestGroupPopulation` | `actor.group_population.read` |
| `GET /groups/{group_id}/population/history` | `ListGroupPopulations` | `actor.group_population.list` |

The latest figure is the one with the latest `as_of`; among figures of the
same date, the most recently recorded. The history is oldest date first. A
group with no figure answers `ErrPopulationNotFound` (404); a negative count,
a missing or malformed date, or an empty source is `ErrInvalidPopulation`
(400); an unknown group is `ErrGroupNotFound`.

## Not decided here

- The coverage figure itself (members ÷ population) is not computed. Which
  members count (direct only, or the group's sub-groups too) and whether a
  sub-group's populations roll up into its parent's are product questions.
- Who may record a figure is the host's grant on
  `actor.group_population.record`.
- Figures are never corrected or deleted; a wrong one is superseded by a new
  figure. Whether an operator needs to retract one is open.
