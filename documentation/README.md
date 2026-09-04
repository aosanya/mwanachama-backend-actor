# mwanachama-backend-actor — documentation

## Layout

Four folders, in SDLC order, and everything lives under one of them.

| Folder | What's inside |
|--------|---------------|
| [1. requirements/](1.%20requirements/) | Problem, vision and scope for Member/Group management extracted out of the gateway. |
| [2. design/](2.%20design/) | The Member/Group graph schema and how it maps onto `mwanachama-backend-shared`'s entity-graph store. |
| [3. implementation/](3.%20implementation/) | The work: `todo.md` (open board), `todo_done.md` (completed rows + board context). |
| [4. qa/](4.%20qa/) | Test coverage and results. |

## Boards and status

| File | What it holds |
| --- | --- |
| [todo.md](3.%20implementation/todo.md) | Open task board |
| [todo_done.md](3.%20implementation/todo_done.md) | Completed rows + board context |

## What this repo is

Extracts `mwanachama-backend-api-gateway`'s `member` domain (and `chapter`,
renamed `Group`) onto the `mwanachama-backend-taskmanager` pattern — domain
logic and storage both delegated to
[mwanachama-backend-shared](../mwanachama-backend-shared)'s entity-graph
store, and imported directly by
[mwanachama-backend-api-gateway](../mwanachama-backend-api-gateway) — no
gRPC, no sub-service shape. Decided in a dev-research session, 2026-09-03
(DSN-1698/DSN-1699 on the gateway's `documentation/2. design/todo.md`).
