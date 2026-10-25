# Areas: a group names its area node

ACT75, member app decisions 495 and 497 (owner, 2026-10-25). Supersedes ACT32
and ACT70: a place's outline, its streets and the point and address lookups
belong to the digital twin (digitaltwin W28), not to this module. Actor keeps
only which twin node a group is.

## What is built

- **A level may be an area.** `level.is_area` is a bool the template sets. The
  level keeps the template's own name (District, Ward, Branch, Site); this
  module never names the rung, it only knows the rung is an area.
- **A group at an area level names its area node.** `group.area_node_id` is a
  plain string: the id of the group's node in the digital twin. This module
  stores it and never resolves it. It is set on `CreateGroup` or with
  `SetGroupAreaNode`, and read on every group read. A group whose level is not
  an area refuses a non-empty id with `ErrNotAnArea` (409), on both paths. An
  empty id clears it and is always accepted.
- **Declared operation.** `set_group_area_node`, `PUT
  /groups/{group_id}/area-node`, action `actor.group.set_area_node`, tool
  `actor_set_group_area_node`; body `area_node_id`, omitted to clear.
- **Upgrade path.** `area_node_id` and `is_area` are in
  `columnsDeclaredSince`, so `Provision` adds them to tables created before
  them. An old `outline` column on an existing table is left where it is and
  no longer read or written.
- **Reach and coverage keep counting by group.** Nothing here reads the twin.

## Retired

- `group.outline`, `SetGroupOutline`, `ErrInvalidOutline` and the
  `set_group_outline` operation (`PUT /groups/{group_id}/outline`).
- `ResolvePoint` and the `resolve_point` operation
  (`GET /hierarchies/{hierarchy_id}/resolve-point`).
- `models.FoldStreetName`.

None of these had a caller outside this repo when they were removed (grepped
across the sibling repos, 2026-10-25).

## Open

- Marking or unmarking an existing level an area after creation (owner).
