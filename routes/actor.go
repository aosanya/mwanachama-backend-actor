// actor.go — HTTP routes over actor_impl.go's plain Actor CRUD:
// CreateActor, GetActor, ListActors, SetActorDisplayName. See doc.go for
// what is and is not in scope and why — these are the identity-only
// operations with no visibility fencing, contact redaction or bulk-lookup
// leak guard layered on top; the gateway's own getMember/lookupMembers keep
// that layering and are NOT replaced by these.
package routes

import (
	"errors"
	"net/http"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

// actorStatusFor maps this package's Actor error sentinels to a status
// code — the same shape statusFor (group.go) uses for Group.
func actorStatusFor(err error) int {
	switch {
	case errors.Is(err, mwanachamaactor.ErrActorNotFound):
		return http.StatusNotFound
	case errors.Is(err, mwanachamaactor.ErrInvalidActor),
		errors.Is(err, mwanachamaactor.ErrDuplicateAttribute):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeActorErr(w http.ResponseWriter, err error) {
	code := actorStatusFor(err)
	if code == http.StatusInternalServerError {
		writeErr(w, code, "internal error")
		return
	}
	writeErr(w, code, err.Error())
}

// CreateActor handles POST — decode, create, encode. No visibility fencing
// and no capability gate: whether a caller may mint an identity row at all
// is the mounting process's own policy (the gateway's own createMember,
// M-1288, keeps CapStructureWrite) — this handler answers only "given that
// a caller may act, is the row valid".
func CreateActor(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in mwanachamaactor.Actor
		if err := readJSON(r, &in); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := um.CreateActor(r.Context(), in)
		if err != nil {
			writeActorErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

// GetActor handles GET {actorID} — the plain, unredacted record. The
// mounting process's own route pattern must name the id path segment
// "actorID". Unlike the gateway's getMember, this carries no co-member
// visibility check and no phone/email redaction (DEV-1128) — those depend
// on domains (Role, capability) this package does not import; a mounting
// process that needs them wraps this handler, or keeps its own, the way
// the gateway still does.
func GetActor(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.GetActor(r.Context(), r.PathValue("actorID"))
		if err != nil {
			writeActorErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ListActors handles GET — every non-deleted actor, unfiltered. No paging,
// no visibility fencing: matches ListActors' own contract. A mounting
// process exposing this to any caller narrower than "every actor in the
// organization is fine to enumerate" needs its own gate in front of it —
// the gateway's own GET /v1/members (member_directory_handlers.go) is that
// gate, composed over more than this package owns, and does not call this
// handler.
func ListActors(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := um.ListActors(r.Context())
		if err != nil {
			writeActorErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// SetActorDisplayName handles PATCH {actorID}/display-name — the one field
// an actor's own record exposes as a distinct write, mirroring
// UserManager.SetActorDisplayName exactly (whole-string replace, no partial
// edit). Same "actorID" path-value requirement as GetActor.
func SetActorDisplayName(um mwanachamaactor.UserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DisplayName string `json:"display_name"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := um.SetActorDisplayName(r.Context(), r.PathValue("actorID"), body.DisplayName)
		if err != nil {
			writeActorErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
