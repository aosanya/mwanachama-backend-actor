# A member's own Forms week split

ACT24, from the member app's decision 196 (owner, 2026-10-21). The same shape
as [group-priority.md](group-priority.md).

## What is stored

The actor record carries `forms_week_split`, a bool declared on the
blueprint's `actor` object with `true` as its default. On, the member's
waiting forms are split into those closing this week and those closing later;
off, they are one list. It is the member's own choice, kept on their actor row
in the agency, so it is per member per instance and follows them to any
device. What "this week" means and how the list is drawn are the member app's.

## Every actor starts with it on

`insertActor`, which `CreateActor` and `EnsureActor` both go through, writes
`true`. A row written before the column existed reads back `true`: `Provision`
adds the column to an existing table through `columnsDeclaredSince` as
`boolean default true`. Changing an actor's name or display name leaves it
alone, since those writes name only their own columns.

## Only the member reads and sets it, and only their own

- `GET /actors/mine/forms-week-split` (`actor.actor.read_my_forms_week_split`,
  `GetMyFormsWeekSplit`, tool `actor_get_my_forms_week_split`) answers
  `{"actor_id", "forms_week_split"}` for the caller. An actor with no record
  yet reads `true`, the default it would be given.
- `PUT /actors/mine/forms-week-split` (`actor.actor.set_my_forms_week_split`,
  `SetMyFormsWeekSplit`, tool `actor_set_my_forms_week_split`) takes
  `{"forms_week_split": bool}`. An actor with no record, and a call with no
  caller, answer `ErrActorNotFound` (404).

Both bind the actor `from: caller`, so a body naming another actor is ignored
and nobody can read or set somebody else's choice. `GET /actors/mine` returns
the field too, as part of the actor record.
