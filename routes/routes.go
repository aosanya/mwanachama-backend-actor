package routes

import (
	"net/http"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

// Route is one address this package answers, relative to wherever the
// mounting process prefixes it (e.g. "/v1/member") — enough to build one
// *http.ServeMux entry from, without the mounting process hand-spelling
// each path/method pair itself. Handler is what CreateGroup/EditGroup/
// MoveGroup and their Actor/ActorGroupAssignment siblings already return:
// decode/call UserManager/encode, no caller gate — see doc.go for why that
// gate is deliberately not this package's to add. Path uses net/http's
// ServeMux pattern syntax ("{groupID}" etc.), so the mounting process only
// ever needs prefix+rt.Path, never its own copy of the path text.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Pattern returns the http.ServeMux registration pattern for this route
// once mounted under prefix — r.Method+" "+prefix+r.Path, net/http's own
// "METHOD /path" syntax (Go 1.22+ mux patterns). A convenience so a caller
// ranging over a route list need not restate the join.
func (r Route) Pattern(prefix string) string {
	return r.Method + " " + prefix + r.Path
}

// ResourceNames configures the URL noun each type is addressed under.
// mwanachama-backend-actor has no notion of what any particular mounting
// process calls Actor/Group/ActorGroupAssignment in its own domain
// (DSN-1699 gap 2 — "the org-facing label is the gateway's call, never
// this package's" — applies to a URL segment exactly the way it applies to
// a display name): every field here defaults to this package's own noun
// for the type when left "", and a caller overrides only the ones its own
// domain renames. The gateway, for example, overrides Group to "chapters"
// and leaves Actor/Assignment at their defaults.
type ResourceNames struct {
	// Actor names the Actor resource segment. Default "actors".
	Actor string
	// Group names the Group resource segment. Default "groups" — the
	// gateway overrides this to "chapters".
	Group string
	// Assignment names the ActorGroupAssignment resource segment, nested
	// under Actor or Group (e.g. "/actors/{actorID}/assignments"). Default
	// "assignments".
	Assignment string
}

// withDefaults fills in every empty field's default noun.
func (n ResourceNames) withDefaults() ResourceNames {
	if n.Actor == "" {
		n.Actor = "actors"
	}
	if n.Group == "" {
		n.Group = "groups"
	}
	if n.Assignment == "" {
		n.Assignment = "assignments"
	}
	return n
}

// GroupRoutes is the three Group writes CreateGroup/EditGroup/MoveGroup,
// addressed under names.Group (or its default, "groups"). A mounting
// process builds its mux from it directly:
//
//	for _, rt := range routes.GroupRoutes(um, hc, routes.ResourceNames{}) { // -> /groups...
//	    mux.HandleFunc(rt.Pattern(prefix), authWrap(rt.Handler))
//	}
//
//	for _, rt := range routes.GroupRoutes(um, hc, routes.ResourceNames{Group: "chapters"}) {
//	    mux.HandleFunc(rt.Pattern(prefix), authWrap(rt.Handler)) // -> /chapters...
//	}
//
// Every route here needs the same caller gate today — "may this caller
// write the org structure" — so a mounting process wraps every Handler the
// same way; a future route with a different gate would need its own field
// on Route to say so, not a change to this loop shape. See doc.go for why
// this package answers none of that itself.
func GroupRoutes(um mwanachamaactor.UserManager, hc HierarchyChecker, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Group
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateGroup(um, hc)},
		{Method: http.MethodPatch, Path: base + "/{groupID}", Handler: EditGroup(um, hc)},
		{Method: http.MethodPost, Path: base + "/{groupID}/move", Handler: MoveGroup(um)},
	}
}

// ActorRoutes is the four plain Actor operations CreateActor/GetActor/
// ListActors/SetActorDisplayName, addressed under names.Actor (or its
// default, "actors"). None of these carry the gateway's own visibility
// fencing or contact redaction — see actor.go's per-handler docs.
func ActorRoutes(um mwanachamaactor.UserManager, names ResourceNames) []Route {
	names = names.withDefaults()
	base := "/" + names.Actor
	return []Route{
		{Method: http.MethodPost, Path: base, Handler: CreateActor(um)},
		{Method: http.MethodGet, Path: base, Handler: ListActors(um)},
		{Method: http.MethodGet, Path: base + "/{actorID}", Handler: GetActor(um)},
		{Method: http.MethodPatch, Path: base + "/{actorID}/display-name", Handler: SetActorDisplayName(um)},
	}
}

// ActorGroupAssignmentRoutes is the five plain ActorGroupAssignment
// operations AssignGroup/Deregister/ListGroupsForActor/ListActorsForGroup/
// HomeCounts, addressed under names.Actor/names.Group (for whose id the row
// nests under) and names.Assignment (default "assignments") for the
// relationship segment itself. See assignment.go's package doc for what
// gateway-only policy these do NOT carry.
func ActorGroupAssignmentRoutes(um mwanachamaactor.UserManager, names ResourceNames) []Route {
	names = names.withDefaults()
	actorBase := "/" + names.Actor
	groupBase := "/" + names.Group
	assignment := names.Assignment
	return []Route{
		{Method: http.MethodPost, Path: actorBase + "/{actorID}/" + assignment, Handler: AssignGroup(um)},
		{Method: http.MethodDelete, Path: actorBase + "/{actorID}/" + assignment + "/{groupID}", Handler: Deregister(um)},
		{Method: http.MethodGet, Path: actorBase + "/{actorID}/" + assignment, Handler: ListGroupsForActor(um)},
		{Method: http.MethodGet, Path: groupBase + "/{groupID}/" + assignment, Handler: ListActorsForGroup(um)},
		{Method: http.MethodGet, Path: "/" + assignment + "/home-counts", Handler: HomeCounts(um)},
	}
}

// Routes is every address this package answers today: GroupRoutes,
// ActorRoutes and ActorGroupAssignmentRoutes concatenated, sharing one
// ResourceNames so the three stay consistent (e.g. Group's "chapters"
// override is honoured in ActorGroupAssignmentRoutes' /{groupID}/assignments
// path too). A mounting process that wants all of it in one loop uses this;
// one that wants to wrap Group's writes differently from Actor's (the
// gateway does, today — CapStructureWrite is Group-specific) calls the
// three functions above separately instead.
func Routes(um mwanachamaactor.UserManager, hc HierarchyChecker, names ResourceNames) []Route {
	out := GroupRoutes(um, hc, names)
	out = append(out, ActorRoutes(um, names)...)
	out = append(out, ActorGroupAssignmentRoutes(um, names)...)
	return out
}
