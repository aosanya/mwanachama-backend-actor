# A manager for one test run (HS280)

`UserManager.ForRun(runID)` returns a manager that reads and writes only the
`_test` copy of every table, under one run id. The parent is not changed, and
an empty run id is refused with `specstore.ErrRunIDEmpty`. The host removes a
run's rows with the shared engine's `RemoveRun` and counts them with
`RowsOfRun`.

## What the copy swaps

- The store: the copy holds `store.ForRun(runID)`, so `store.Query`, `Take`,
  `Insert` and `List` hit the `_test` table and filter on `run_id`.
- Raw queries: every `tx.Table(...)` in the module goes through
  `userManager.from(tx, role)`, which names the `_test` table and adds
  `run_id = ?`. Sub-queries, transactions and the leader, outsider, invitation
  and join helpers all take it from there.
- Inserts built by hand: rows are made with `store.Row(role, value)`, which
  stamps `run_id`.
- Code counters: `nextCode` reads, claims and seeds the `code_sequence` row of
  that run only, so each run counts its group, actor and level codes from one.
  Public-code collision checks and the code format read the same way.

## What stays shared

The code-group setting resolves from the live settings layers, as the live
manager does. The invitation lifetime is copied when `ForRun` is called.

## Tests

`testrun_test.go`: a run leaves no live row, a live write leaves the copy
empty, two runs never see each other, each run mints its own codes, a change in
one run leaves another run's rows alone (including a level delete that counts
groups), removing a run keeps the others, and an empty id is refused.
