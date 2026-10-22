# Areas: a group that carries a place

ACT32, member app decisions 301–303 (2026-10-22). This page records the slice
built so far and what is still open on the row.

## What is built

- **A level may be an area.** `level.is_area` is a bool the template sets. The
  level keeps the template's own name (District, Ward, Branch, Site); this
  module never names the rung, it only knows the rung is an area.
- **A group at an area level may carry an outline.** `group.outline` is a
  shared S65 `geometry` field (`geo.Geometry`), stored as jsonb on Postgres and
  text on SQLite. An outline must be a `Polygon` or a `MultiPolygon` that
  passes `geo.Validate`; anything else is `ErrInvalidOutline`. A group whose
  level is not an area refuses one with `ErrNotAnArea`, both on `CreateGroup`
  and on `SetGroupOutline`. An empty geometry is no outline and is stored as
  NULL.
- **`ResolvePoint(ctx, hierarchyID, lon, lat)`** answers the group whose
  outline contains the pin, among groups at an area level of that hierarchy,
  as `(group, found, err)`. A hierarchy with no area level answers none, as
  does a pin outside every outline. When several area levels nest and more
  than one outline contains the pin, the deepest level wins, since that is
  the finer answer. Containment is `geo.Contains`, so a pin on the boundary
  counts as inside.
- **Upgrade path.** `outline` and `is_area` are in `columnsDeclaredSince`, so
  `Provision` adds them to tables created before ACT32. A geometry column is
  added without a default, NULL meaning no outline.

## Declared operations

`SetGroupOutline` and `ResolvePoint` are declared in `actor.operations.json`
like every other address, so each is a route and an MCP tool from one entry:

| Operation | Address | Action | Tool |
| --- | --- | --- | --- |
| `set_group_outline` | `PUT /groups/{group_id}/outline` | `actor.group.set_outline` | `actor_set_group_outline` |
| `resolve_point` | `GET /hierarchies/{hierarchy_id}/resolve-point?lon=&lat=` | `actor.area.resolve_point` | `actor_resolve_point` |

The outline is the body's `outline` field as GeoJSON; a body without one
clears it. `ErrNotAnArea` answers 409 (the group's level is the conflict, not
the request's shape) and `ErrInvalidOutline` 400. `resolve_point` answers
`{group, found}`, the same two-value shape `deregister` uses, so a miss is a
200 with `found: false` rather than a 404: a pin in no area is an answer, not
a missing thing. An unknown hierarchy answers the same miss, since
`ResolvePoint` lists its levels and finds none. Marking a level an area needed
no new address: `create_level` takes the whole level, `is_area` included.
`routes/area_test.go` walks all three through the real table.

## Why the walk is in Go

`ResolvePoint` lists the hierarchy's groups and tests each outline in Go,
the same posture `MoveGroup`'s cycle check takes. An agency has tens to a few
hundred areas, and SQLite has no spatial index to lean on in tests. A
deployment large enough to feel this is where a bounding-box prefilter, or
PostGIS, earns its place.

## Still open on the row

- **The street list** on a group at an area level (street name, house-number
  ranges, odd/even/all), and **`ResolveAddress`**, which matches ignoring
  case, accents and common abbreviations such as "u." for "utca".
- **Routes and MCP tools for `ResolveAddress`**, which waits on the street
  list.
- **Marking an existing level an area.** A level is marked only when it is
  created (`POST /levels` with `is_area`); `RenameLevel` touches the name
  alone and there is no address that flips `is_area` afterwards. Whether a
  template's level may become an area, or stop being one, after groups wear
  it is the owner's call: un-marking would strand every outline on it.
- `models/schema_generated_test.go` was edited by hand to take `IsArea` and
  `Outline`. The `structure.json` its banner names is not in the developer
  repo: `developer/tools/domain-schema`'s Makefile reads it from
  `mwanachama-backend-api-gateway/internal/domain/structure/`, which is
  retired. A regeneration from that file would drop both fields, so the file
  stays hand-edited until the generator is pointed at a live source.

## The municipality template

`developer/documentation/4. qa/agencies/municipality/modules/council_actor.json`
marks its Ward and Neighbourhood levels `is_area` and leaves Municipality and
Function unmarked: the first two are places, the other two are the elected
body and its offices. Its Ward and Neighbourhood groups are made at
provisioning from the municipality's own boundaries, so the template names
where each outline comes from and carries no geography itself.
