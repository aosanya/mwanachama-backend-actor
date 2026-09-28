# ACTD-001 — The domain is declared, not written: actor adopts catalog's blueprint/spec/operations format

Scoped 2026-09-28, prompted by the user pointing at
`mwanachama-backend-catalog` and asking for this repo to be put "in the same
format". This is the record the implementation rows **ACT5–ACT12** point at;
it decides nothing that those rows cannot revisit, and it deliberately leaves
three questions open (#2, #6, #9 below) because they are the owner's, not a
session's.

## What the format is

Not restated here. `mwanachama-backend-shared` owns it:

- [declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md)
  — blueprint, domain spec, `spec.Migrate`, `specstore`, and the six-step
  "Converting a repo" order ACT5–ACT12 follow.
- [dispatcher.md](../../../mwanachama-backend-shared/documentation/2.%20design/dispatcher.md)
  — declared operations to `httpwire.Route`s and MCP tools, deny-by-default
  routing and gating.

`mwanachama-backend-catalog` invented it and is the reference;
`mwanachama-backend-agency` is the worked conversion (AGD-007 on its own
board, landed as AG37–AG43 on 2026-09-24 — 18 objects, 64 operations,
`gormstore/` deleted). This repo is the third, and the first whose consumer
does not mount the module's own route table.

## What is here today

| | today | after |
| --- | --- | --- |
| storage | `gormstore/`, 9 row structs, 63 fields, `AutoMigrate` per table | `actor.blueprint.json` + `actor.actor.json`, `spec.Migrate` |
| tables | `<instance>_<table>` — `actor_actors` | `<instance>_actor_<table>` |
| the route table | `routes/`, 16 addresses hand-written over 49 manager methods | `actor.operations.json` + `dispatch` |
| gating | none — `routes.Route` has no action | an action per operation, gated at the mount |
| MCP | none | `routes/tools.go` from the same operations file |
| codes | `gormstore/codesequence.go`'s `NextCode` | `nextCode` in `store.go` |

## Decisions

1. **Actor converts completely, not just its storage.** Nine objects —
   actor, group, actor_group_assignment, role_kind, actor_role_assignment,
   hierarchy, level, group_type, code_sequence — are declared;
   `actor.operations.json` declares the API; `routes/` becomes `routes.go` +
   `tools.go` over `dispatch`. `gormstore/` is deleted. `models/` survives as
   carrier structs, which is what the store reads rows into.
   `GroupDashboard` stays hand-written: it is a read model over four objects,
   not an object.

2. **Open — the instance segment.** `<instance>_<module>_<table>` with
   today's instance gives `actor_actor_actors`, which is agency's own
   `agency_agency_goals` shape and needs nothing from anybody. The
   alternative is renaming the instance, but the gateway's `UserInstance.Name`
   is *also* its route prefix (`/v1/actor`), so a rename moves published URLs
   as well as tables. Recommendation: keep `actor`, take the doubled segment.

3. **`Attributes` is a declared `json` field, and its uniqueness becomes
   declared indexes.** `models.Property`'s `Unique` half — today
   `gormstore.syncUniqueAttributeIndexes` plus the root package's
   `checkUniqueAttributes` pre-check — is exactly a unique document-path
   index in the domain spec (`{"path": {"field": "attributes", "path":
   "phone"}, "unique": true}`), and `spec`'s `DocPathExpr` already emits the
   `attributes ->> 'phone'` / `json_extract` pair per dialect. Required,
   Range and Options stay in Go, in `models.ValidateAttributes`, where a
   caller may still pass its own longer property list.

   One semantic difference to carry deliberately: the pre-check skips a value
   that is `nil` or `""`, so two rows may both hold `attributes.phone = ""`
   today. A declared unique index does not skip it — absent and SQL NULL stay
   non-colliding in both dialects, but two empty strings become a collision.
   Either keep writing `""` as an absent key, or accept the refusal.

4. **Sequences stay in Go, rewritten rather than ported.**
   declared-domains.md's own "Not solved here" says nothing declares a
   per-type counter and names agency as the first module to need one — actor
   is the second, which is the condition that section sets for moving it into
   `mwanachama-backend-shared`. Either way the code that lands is agency's
   `store.go` `nextCode`: a CAS write retried on `RowsAffected == 0`, not
   `gormstore/codesequence.go`'s read-then-write. That is what closes ACT3,
   the same way AG39's rewrite closed AG34/AG35.

5. **`is_default_anchor` has no declared equivalent, and must not get one by
   the back door.** `spec.Index` offers `unique` and `not_deleted` and
   nothing else, on purpose — declared-domains.md refuses free SQL in an
   index condition because it would be an injection surface with no upside.
   The partial unique index behind "at most one default-anchor level per
   hierarchy" (`WHERE is_default_anchor = TRUE`, the gateway's retired
   migration 000033 guarantee) is therefore either a second named condition
   taught to `spec`, or a hand-written statement in this repo's `Provision`.
   Decided at ACT7, not here. `CreateLevel`/`SetDefaultAnchor` enforce the
   same rule in Go regardless, so this is the database-level half only.

6. **Open — what the declared surface publishes.** There are 49 manager
   methods and 16 declared addresses. Hierarchies, levels, group types, role
   kind retirement, grants and revocations and the dashboard have no HTTP
   route in this repo at all: the gateway hand-writes its own handlers for
   them. Declaring the union is what agency did, and it gave the gateway four
   new routes and its Postman gate four new uncovered ones; here it would be
   about thirty. Recommendation: declare the operations the gateway already
   serves plus the ones this repo's own `routes/` serves, and add the rest
   when a mount asks — a declared operation nobody serves is still a route
   somebody must cover. `HierarchyExists`/`LevelInHierarchy` are predicates
   the route layer needs, never addresses.

7. **`ResourceNames` goes away.** Nothing in the fleet sets it — grepped
   across every sibling repo. Its stated purpose (a mount addressing Group
   under its own noun) is real, and the gateway does exercise it, but not
   through this type: it mounts `/structures` paths of its own around
   `routes.CreateGroup`/`EditGroup`/`MoveGroup` as bare
   `http.HandlerFunc`s. Declared paths are literal, so there is no override
   to carry over.

8. **The gateway keeps its own nouns for the cutover.** `dispatch` emits
   `httpwire.Route` values, not the per-handler constructors the gateway
   mounts today, so the gateway either adopts the declared paths — moving its
   published `/v1/actor/structures*` API and every Postman request behind it
   — or keeps its hand-written handlers and calls `UserManager` directly.
   Take the second: this conversion should not move a published gateway URL.
   Adopting the declared table is then a gateway decision on the gateway's
   own board.

9. **Open — the MCP tool surface.** Catalog's `routes/tools.go` derives one
   from the operations file for about forty lines of carrier and
   store-owned-field maps, so having declared the operations, not having
   tools costs more explanation than having them. Against: this repo's own
   CLAUDE.md says an in-library `mcp/` is a *may*, reached for when a domain
   has exactly one hosting product, and nothing hosts actor that way today.
   Recommendation: ship `tools.go`, mount it nowhere until a host asks.

10. **The gateway's SQL mirror is regenerated, and it is already wrong.**
    `internal/store/postgres/migrations/000001_actor_tables.up.sql` creates
    four tables under the `member_` prefix — `member_actors`,
    `member_groups`, `member_actor_group_assignments`,
    `member_code_sequences` — while the mounted instance has been `actor`
    since the rename, and no active migration creates `role_kinds`,
    `actor_role_assignments`, `hierarchies`, `levels` or `group_types` at
    all. So `cmd/migrate up` cannot provision a fresh database for this
    module today, which is the whole point of the mirror. `spec.DDL("postgres")`
    emits the replacement.

## What this costs, and what it does not buy back

- **A third declaration of the same fields already exists.**
  `developer/tools/domain-schema` generates the gateway's
  `internal/domain/{actor,role,structure}` types *and* this repo's
  `models/schema_generated_test.go`, which pins every carrier's field order
  and json tags by reflection. After the conversion a field lands in the
  blueprint (storage), in `models/` (the carrier), and in the gateway's
  `*.json` (the shared shape) — three files, one of which fails loudly.
  Nothing merges them here; ACT11 records the order to edit them in.
- **Every route arrives gated for the first time.** Actor's `routes.Route`
  carries no action, so the mounting process is the only gate today. The
  declared table carries `actor.<resource>.<verb>` per operation, which is a
  new refusal path in the gateway rather than a free win.
- **A retirement is still a migration somebody writes.** `spec.Migrate` only
  ever adds, so the pre-rename tables survive whatever `Provision` does not
  explicitly rename.
- **ACT3 and ACT4 close as side effects, not as fixes of their own.** The
  store rewrite replaces the code both rows point at. ACT4's two pinning
  tests in `routes/createserverfields_test.go` must go red the moment
  `Create*` stops honouring a caller-supplied `id`, `created_at` or
  `deleted`; `GetActor`'s missing `deleted = false` filter is the other half
  of that row and becomes a `not_deleted` read.
