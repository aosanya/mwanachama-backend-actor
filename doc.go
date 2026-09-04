// Package mwanachamaactor provides Actor/Group lifecycle management,
// extracted from mwanachama-backend-api-gateway's internal/domain/member and
// internal/domain/chapter packages onto GORM-backed relational storage:
// domain logic AND storage both live in this package, imported directly
// into the gateway process — no separate service, no gRPC, no proto.
// Decided in a dev-research session, 2026-09-03 (DSN-1698); moved off
// mwanachama-backend-shared/entitygraph onto GORM once the org moved away
// from the graph-storage model entitygraph provided.
//
// Layout:
//   - models/    — domain types (Actor, Group, GroupEdit, ActorGroupAssignment);
//     callers use models.Actor etc. directly, no re-export in this package
//   - gormstore/ — GORM row structs, row<->domain conversion, migration
//   - doc.go (this file), tables.go — table-name/migrate wrappers
//   - user.go               — UserManager interface, userManager struct
//   - actor_impl.go         — Actor CRUD
//   - group_impl.go         — Group CRUD, tree operations
//   - registration_impl.go  — ActorGroupAssignment operations
//   - errors.go             — sentinel errors
//
// Ported from mwanachama-backend-api-gateway's internal/domain/member and
// internal/domain/chapter packages. See this repo's CLAUDE.md for what
// changed along the way.
package mwanachamaactor
