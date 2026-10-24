# Actor settings

ACT31 (member app decisions 280, 281, 398). The actor blueprint declares its
settings through shared's S64 engine, so a module default, a definition value
and an operator override each have their own layer. See
[declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md)
"A module declares its settings".

## `code_groups`

How many groups of three characters a newly minted public code carries after
its prefix (`M-…` for an actor, `G-…` for a group; see
[public-codes.md](public-codes.md)). Declared as an `int` with default `3`,
which is ACT21's `models.DefaultCodeGroups`, so a deployment with no stored
layer mints exactly what it minted before.

Minting reads the length in this order:

1. The code format row an operator set through `SetCodeFormat` (ACT21's
   address), when it exists and holds at least one group.
2. Otherwise the setting, resolved from the stored layers with
   `spec.ResolveStored`: override, then definition, then the blueprint
   default.

`CodeFormat` answers the same resolved value when no row has been set.
A resolved value outside 1..`models.MaxCodeGroups` refuses the mint rather
than drawing a code of that length; `models.ParseCodeGroups` holds the bound.
Codes already minted keep their length whatever the setting later says.

`spec.Migrate`, called from `Provision`, creates the three settings tables in
the actor's own table set, so nothing else is needed to provision them.

## Left

- **Role kinds' settings.** Nothing on a role kind today is a module-wide
  configurable: its capabilities, `is_delegate`, `leads_group`, `outsider`
  and level are per-row data written through the role kind addresses. No
  setting was invented for them.
- **The code format row and the setting overlap.** Whether `SetCodeFormat`
  should become an override write and the row retire is not decided here.
- **A stale override after a definition is republished** is an open owner
  question and is not decided here: the override keeps winning.
