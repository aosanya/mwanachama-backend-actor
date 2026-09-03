# Design

The Member/Group graph schema (see `../../schema.go`'s `DefaultUserSchema`)
and how it maps onto `mwanachama-backend-shared`'s entity-graph store.

## Topology

```
Member ──registered_at──► Group   (properties: is_home, joined_at)
Group  ──has_member──────► Member (inverse of registered_at)
```

## Decisions carried in from DSN-1698

- Full `mwanachama-backend-taskmanager` pattern: domain logic + storage on
  `entitygraph.DataManager`.
- Physical tables: `postgres.DefaultTableNames("member_")` →
  `member_entities` / `member_relationships` / `member_schemas_draft` /
  `member_schemas_published`.
- `Chapter` moves as the generic type `Group`. The gateway is responsible
  for presenting it under an org-configurable label — this repo never
  hardcodes "Chapter".
- `Registration` is the `registered_at` relationship, not a table.
- "One home chapter per member" is dropped outright — `is_home` carries no
  exclusivity anywhere in this repo.

## Defaults taken for DSN-1699's unresolved gaps

- **Gap 1 (hierarchy / chapter_level fate):** DEFAULT taken — `hierarchy`
  and `chapter_level` stay physically in the gateway's own Postgres tables.
  `Group` carries `hierarchy_id` / `level_id` / `parent_id` as plain JSONB
  properties with **no foreign key** back to those tables. This means
  `Level.IsDefaultAnchor` / `Chapter.AnchorLevelOverrideID`'s cross-hierarchy
  composite-FK guarantee (gateway migration 000034) has **no equivalent**
  here — an `anchor_level_override` naming a level from the wrong hierarchy
  is not rejected by this package. See `schema.go`'s package doc for the
  same note in code.
- **Gap 2 (org-facing label):** not addressed by this repo at all — no new
  label infrastructure was built. `Group.Name`/`Group.NodeType` are the only
  presentation-adjacent fields, and nothing here reads or writes a
  per-org "Chapter" label.
- **Gap 3 (chapter's satellite features):** dashboard rollups,
  `rpc_discoverable_chapters`-equivalent discoverability, and
  `chapter_membership_cap` stay gateway-side, composed over this package's
  `ListDiscoverableGroups`/`ListGroups` the way `Directory`/
  `AddressDirectory` already compose across stores today. This repo exposes
  the primitives; it does not compose them.
- **Gap 4 (route/test parity):** unaffected by this repo directly — the
  gateway's `member.Repository`/`chapter.Repository` interfaces are
  unchanged; parity is a gateway-side adapter concern.
