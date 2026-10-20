# An actor's names

Written 2026-10-20 for wakala-api's WK116, on the member app's decisions 91
and 94: a member has a first name and a last name beside the name they are
shown by.

## Three names, one of them shown

| Field | What it is |
|---|---|
| `display_name` | What to call the actor on screen, and the only name the module promises to show. A host may call it a screen name. Set by `SetActorDisplayName` (`PATCH /actors/{actor_id}/display-name`). |
| `first_name` | The given name, as the actor gave it. |
| `last_name` | The family name, as the actor gave it. |

`first_name` and `last_name` are set together by `SetActorName(ctx, id,
first, last)`, which writes both and `last_updated` and touches nothing else,
so renaming the display name never moves them and setting them never moves
the display name. Either may be empty; `""` means not given. The module does
not decide who may read them: which callers see the first and last name and
which only the display name is the host's to say. No operation is declared
for `SetActorName`, so the actor mount exposes no route for it; a host calls
it in Go, as wakala-api's `PUT /instances/{id}/me/name` does.

An existing `actor` table gains both columns on the next `Provision`
(`columnsDeclaredSince`), and every row reads them as `""`.

## What the Actor type carries and why

Phone and email are not columns. They are `attributes` entries declared by
`DefaultActorProperties`, so the built-in pair and anything an organization
adds later go through the same required, unique and range validation.
`deleted` is a soft delete with no method that sets it; every listing filters
it out. `is_agentic` marks an actor driven by a model rather than a person.
`code` is minted once at creation and never edited.
