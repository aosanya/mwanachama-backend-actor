# Self-corrected attributes

ACT33, from the municipality template's unenforceable wish (AGX14, member app
decision 283): a person may correct their own register record, and a value
they corrected must be told apart from one the council entered.

## Two addresses, one rule

| Operation | Address | Action | Who is recorded |
| --- | --- | --- | --- |
| `correct_my_attributes` | `PATCH /actors/mine/attributes` | `actor.actor.correct_my_attributes` | `self`, the caller |
| `set_actor_attributes` | `PATCH /actors/{actor_id}/attributes` | `actor.actor.set_attributes` | `staff`, the caller |

The body is `{"attributes": {"<name>": <value>, ...}}`. The actor corrected on
the first address is bound from the caller, never the body; a body naming an
`actor_id` is an unknown field and refused with 400. The editor on the second
is bound from the caller too. Who may call the second is the mount's gate, as
for every address here.

Both go through one path, `setAttributes`:

- Only attributes in the declared catalog (`DefaultActorProperties`: `phone`,
  `email`) are accepted. Anything else is `ErrInvalidActor` (400).
- The named attributes are merged over the held ones; an empty value clears
  one. The merged set is validated with `ValidateAttributes`, exactly as on
  create.
- A unique attribute is checked against every other actor, not this one, so
  keeping one's own value is not a duplicate. The partial unique index stays
  the database's half of that refusal.

## Provenance

`actor.attribute_sources` is a JSON document keyed by attribute name, each
entry `{set_by, set_by_id, set_at}`. `set_by` is `self` or `staff`. A write
replaces the entry for every attribute it names, so a self-correction over a
staff value makes it `self`, and a later staff edit makes it `staff` again.
An attribute with no entry predates the record or was given at create, and
reads as entered by staff. Only the latest setter is kept; the history of
who changed what is custody's to record, not this row's.

The column joins `columnsDeclaredSince`, so an existing table gains it on the
next provision.

## Open for the owner

Which attributes a person may self-correct is not stated anywhere. As built,
a person may correct any declared actor attribute. If a domain needs some
attributes to stay staff-only, that is a per-attribute flag on `Property`
(for example `self_correctable`), and a decision to take before adding it.
