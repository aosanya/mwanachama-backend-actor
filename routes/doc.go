// Package routes is mwanachama-backend-actor's own HTTP surface: decode a
// request, call one [mwanachamaactor.UserManager] method, encode the
// response — the same shape group_impl.go's Go callers already get, just
// reachable from an HTTP mux. It exists so a route's request/response
// shape and its domain logic are authored and reviewed together, in the
// package that owns the domain, rather than reimplemented a second time in
// whichever process happens to mount this package.
//
// Scope, as of 2026-09-06 (DEV-1661 added role.go's four): every operation
// that is, underneath, a single UserManager call with no gateway-only
// policy layered on top — Group's three writes (CreateGroup/EditGroup/
// MoveGroup, gated only by the two hierarchy/level reference questions
// DSN-1699 gap 1 left in the mounting process's own tables, answered by the
// [HierarchyChecker] the caller supplies), the plain Actor CRUD
// (CreateActor/GetActor/ListActors/SetActorDisplayName — see actor.go), the
// plain ActorGroupAssignment operations (AssignGroup/Deregister/
// ListGroupsForActor/ListActorsForGroup/HomeCounts — see assignment.go),
// and the plain RoleKind/ActorRoleAssignment reads and creation
// (CreateRoleKind/ListRoleKinds/GetRoleKind/GetRoleAssignment — see
// role.go). [Routes] returns the whole set as one list of addresses a
// mounting process can range over to build a mux from;
// [GroupRoutes]/[ActorRoutes]/[ActorGroupAssignmentRoutes]/[RoleRoutes]
// return one type's slice at a time, for a mounting process that wraps
// different types in different policy (the gateway does, today).
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
// DEV-1661 added a sixth and seventh reason to the same list, for the same
// exclusion: RetireRoleKind/UnretireRoleKind and GrantRole/RevokeRole/
// StepDownRole each resolve a caller identity and compose an audit-log row
// around the UserManager call (see the gateway's role_act_compose.go, added
// by DEV-1658 for exactly this reason — an extracted package must not
// depend on the custody domain even transitively), GrantRole additionally
// carries the amplification and agentic-seat gates, and StepDownRole its
// own holder-only identity check. ListRoleAssignmentsForGroup/
// ListRoleAssignmentsForActor/GroupDashboard carry
// requireChapterCensusReader/requireMemberVisible, the same DEV-1147/
// DEV-1128 visibility fencing the excluded member/chapter reads above
// already carry. All ten stay gateway-side, calling this repo's
// UserManager methods directly rather than through an HTTP route here.
//
// A route built from this package still needs a caller-identity/capability
// gate wrapped around it before it is safe to serve — this package answers
// "what happens once that gate has passed", never "who may pass it". The
// mounting process supplies that gate the same way it already supplies
// [HierarchyChecker]: by wrapping the http.HandlerFunc this package
// returns, not by this package reaching for a session or a capability
// itself.
package routes
