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
| `withdraws` | set only on a withdrawn record: the id of the figure it withdraws |
| `created_at` | when it was recorded |

A separate object because a figure is dated: a newer estimate is added beside
the old one, never written over it, so the history survives.

## Operations

| Address | Call | Action |
| --- | --- | --- |
| `POST /groups/{group_id}/population` | `RecordGroupPopulation` | `actor.group_population.record` |
| `GET /groups/{group_id}/population` | `LatestGroupPopulation` | `actor.group_population.read` |
| `GET /groups/{group_id}/population/history` | `ListGroupPopulations` | `actor.group_population.list` |
| `POST /groups/{group_id}/population/{population_id}/withdraw` | `WithdrawGroupPopulation` | `actor.group_population.withdraw` |

The latest figure is the one with the latest `as_of`; among figures of the
same date, the most recently recorded. The history is oldest date first. A
group with no figure answers `ErrPopulationNotFound` (404); a negative count,
a missing or malformed date, or an empty source is `ErrInvalidPopulation`
(400); an unknown group is `ErrGroupNotFound`.

## Withdrawing a figure and inner groups (ACT72)

ACT72, 2026-10-25, from member app decisions 423 and 424 (owner,
2026-10-24). The group's figure is the one figure: catalog's copy is retired
(CAT133 there).

- **Withdrawing appends.** A figure is withdrawn by appending a record whose
  `withdraws` names it, with `recorded_by` from the caller, the source
  `withdrawn`, and the withdrawn figure's count and date copied so the record
  reads on its own. Nothing is updated or deleted. The latest read skips the
  withdrawn figure and every withdrawn record; the history keeps both. A
  withdrawn record, an already-withdrawn figure, an unknown id or another
  group's figure is `ErrPopulationNotFound`.
- **A parent with no figure of its own reads its inner groups' sum.** Inner
  groups are found through `parent_id`. For each direct inner group the read
  takes that group's latest standing figure, or, if it has none, its own inner
  groups' sum, recursively; so a figure recorded at any level replaces the sum
  beneath it and nothing is counted twice. Inner groups with no figure
  anywhere beneath them add nothing. If nothing is found the answer is
  `ErrPopulationNotFound`. The summed answer is not stored: it has no `id`,
  its source is `inner groups`, and its `as_of` is the oldest date among the
  parts, so the figure never claims to be fresher than its stalest part.
- A group with a standing figure of its own reads that figure even when its
  inner groups' figures sum to more.

## Not decided here

- The coverage figure itself (members ÷ population) is still not computed
  here. Decision 423 settles that it counts a group's members with its inner
  groups' members; the read that divides is not built yet.
- Who may record a figure is the host's grant on
  `actor.group_population.record`.
- Figures are never corrected or deleted; a wrong one is superseded by a new
  figure or withdrawn (above).
- Who may withdraw is the host's grant on `actor.group_population.withdraw`.
