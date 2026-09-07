// role.go — HTTP routes over role_kind_impl.go's/role_assignment_impl.go's
// plain reads and RoleKind creation: CreateRoleKind, ListRoleKinds,
// GetRoleKind, GetRoleAssignment. See doc.go for what is and is not in scope
// and why.
//
// Deliberately NOT here, mirroring actor.go's own exclusions: RetireRoleKind/
// UnretireRoleKind (the caller resolves an actor identity and composes an
// audit-log row around the call — role_kind_custody_test.go and
// role_act_compose.go are the gateway's shape of that), GrantRole/RevokeRole/
// StepDownRole (same, plus GrantRole's amplification/agentic-seat gates and
// StepDownRole's holder-only identity check), and
// ListRoleAssignmentsForGroup/ListRoleAssignmentsForActor/GroupDashboard
// (gateway-only visibility fencing — requireChapterCensusReader/
// requireMemberVisible — over domains, capability and session state, this
// package must not depend on).
package routes

import (
	"errors"
	"net/http"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

// roleStatusFor maps this package's role-kind/role-assignment error
// sentinels to a status code — statusFor's (group.go) sibling for this type.
func roleStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaactor.ErrRoleKindNotFound),
		errors.Is(err, mwanachamaactor.ErrAssignmentNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaactor.ErrKindHasLiveAssignments),
		errors.Is(err, mwanachamaactor.ErrKindRetired):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeRoleErr(w http.ResponseWriter, err error) {
	code := roleStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateRoleKind handles POST — decode, create, encode. No capability gate:
// whether a caller may define a position at all is the mounting process's
// own policy (the gateway's own createRoleKind keeps CapRoleKindWrite).
func CreateRoleKind(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in mwanachamaactor.RoleKind
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := um.CreateRoleKind(r.Context(), in)
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// ListRoleKinds handles GET — every role kind, unfiltered. No visibility
// fencing: role kinds are the organization's defined positions, not
// per-caller data.
func ListRoleKinds(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.ListRoleKinds(r.Context())
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// GetRoleKind handles GET {kindID}. The mounting process's own route
// pattern must name the id path segment "kindID".
func GetRoleKind(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.GetRoleKind(r.Context(), r.PathValue("kindID"))
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// GetRoleAssignment handles GET {assignmentID} — one seat by id, active or
// not, unredacted. Unlike the gateway's own uses of this read (which resolve
// a chapter from it before an authorization check — DEV-1080), this handler
// carries no such fencing: it answers only "given that a caller may act, is
// the row valid". The mounting process's own route pattern must name the id
// path segment "assignmentID".
func GetRoleAssignment(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.GetRoleAssignment(r.Context(), r.PathValue("assignmentID"))
		if err != nil {
			writeRoleErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// RoleRoutes is the four plain role operations, addressed under
// names.RoleKind/names.RoleAssignment (or their defaults, "role-kinds" and
// "role-assignments").
func RoleRoutes(um mwanachamaactor.UserManager, names ResourceNames) []Route {
	names = names.withDefaults()
	kindBase := "/" + names.RoleKind
	assignmentBase := "/" + names.RoleAssignment
	return []Route{
		{Method: http.MethodPost, Path: kindBase, Handler: CreateRoleKind(um)},
		{Method: http.MethodGet, Path: kindBase, Handler: ListRoleKinds(um)},
		{Method: http.MethodGet, Path: kindBase + "/{kindID}", Handler: GetRoleKind(um)},
		{Method: http.MethodGet, Path: assignmentBase + "/{assignmentID}", Handler: GetRoleAssignment(um)},
	}
}
