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
- **Routes and MCP tools** for `SetGroupOutline` and the two resolvers; the
  methods exist on `UserManager` but are not declared in
  `actor.operations.json` yet.
- `models/schema_generated_test.go` was edited by hand to take `IsArea` and
  `Outline`; the developer repo's `structure.json`, which generates it, needs
  the same two fields so a regeneration does not drop them.
