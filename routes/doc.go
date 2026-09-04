// Package routes is mwanachama-backend-actor's own HTTP surface: decode a
// request, call one [mwanachamaactor.UserManager] method, encode the
// response — the same shape group_impl.go's Go callers already get, just
// reachable from an HTTP mux. It exists so a route's request/response
// shape and its domain logic are authored and reviewed together, in the
// package that owns the domain, rather than reimplemented a second time in
// whichever process happens to mount this package.
//
// Scope, as of 2026-09-04: every operation that is, underneath, a single
// UserManager call with no gateway-only policy layered on top — Group's
// three writes (CreateGroup/EditGroup/MoveGroup, gated only by the two
// hierarchy/level reference questions DSN-1699 gap 1 left in the mounting
// process's own tables, answered by the [HierarchyChecker] the caller
// supplies), the plain Actor CRUD (CreateActor/GetActor/ListActors/
// SetActorDisplayName — see actor.go), and the plain
// ActorGroupAssignment operations (AssignGroup/Deregister/
// ListGroupsForActor/ListActorsForGroup/HomeCounts — see assignment.go).
// [Routes] returns the whole set as one list of addresses a mounting
// process can range over to build a mux from; [GroupRoutes]/[ActorRoutes]/
// [ActorGroupAssignmentRoutes] return one type's slice at a time, for a mounting
// process that wraps different types in different policy (the gateway
// does, today).
//
// Deliberately NOT here, and not a future TODO — a considered exclusion:
// the gateway's own getMember, register, deregister,
// listRegistrationsForMember and listMembersForChapter, even though each
// has a same-shaped plain counterpart in this package (GetActor,
// AssignGroup, Deregister, ListGroupsForActor, ListActorsForGroup
// respectively). Those five compose
// with domains this package must not depend on — DEV-1287 visibility
// fencing (chapter_visibility.go) and DEV-1128's co-member/contact-redaction
// fencing (member_visibility.go), the membership cap (org_policy), the
// agentic-seat gate, Role/capability checks, and DEV-1343/DEV-1344's
// act-log composition on deregister. Pulling any of those in here would
// mean this package importing the gateway's session/capability/role
// machinery, which the root package's own doc comment already refuses on
// purpose ("no separate service... imported directly"). See
// mwanachama-backend-api-gateway's member_instance_routes.go for what stays
// there and why, and its CLAUDE.md's "chapter-scoped access is enforced
// here, server-side" invariant for the reason it stays there. This
// package's plain routes are not drop-in replacements for those five —
// they are the same undecorated shell Group's three writes already are,
// available for a mounting process to wrap however its own policy demands.
//
// A route built from this package still needs a caller-identity/capability
// gate wrapped around it before it is safe to serve — this package answers
// "what happens once that gate has passed", never "who may pass it". The
// mounting process supplies that gate the same way it already supplies
// [HierarchyChecker]: by wrapping the http.HandlerFunc this package
// returns, not by this package reaching for a session or a capability
// itself.
package routes
