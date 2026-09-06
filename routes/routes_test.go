package routes_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func patterns(rts []routes.Route, prefix string) []string {
	out := make([]string, len(rts))
	for i, rt := range rts {
		out[i] = rt.Pattern(prefix)
	}
	return out
}

func assertPatterns(t *testing.T, got []routes.Route, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d routes, want %d: %v", len(got), len(want), patterns(got, ""))
	}
	for i, p := range patterns(got, "") {
		if p != want[i] {
			t.Fatalf("route %d: got %q, want %q", i, p, want[i])
		}
	}
}

func TestGroupRoutes_DefaultResourceIsGroups(t *testing.T) {
	um := newTestManager(t)
	rts := routes.GroupRoutes(um, stubHierarchy{}, routes.ResourceNames{})
	assertPatterns(t, rts, []string{"POST /groups", "PATCH /groups/{groupID}", "POST /groups/{groupID}/move"})
}

func TestGroupRoutes_ResourceOverride(t *testing.T) {
	um := newTestManager(t)
	rts := routes.GroupRoutes(um, stubHierarchy{}, routes.ResourceNames{Group: "chapters"})
	assertPatterns(t, rts, []string{"POST /chapters", "PATCH /chapters/{groupID}", "POST /chapters/{groupID}/move"})
}

func TestActorRoutes_DefaultResourceIsActors(t *testing.T) {
	um := newTestManager(t)
	rts := routes.ActorRoutes(um, routes.ResourceNames{})
	assertPatterns(t, rts, []string{
		"POST /actors",
		"GET /actors",
		"GET /actors/{actorID}",
		"PATCH /actors/{actorID}/display-name",
	})
}

func TestActorRoutes_ResourceOverride(t *testing.T) {
	um := newTestManager(t)
	rts := routes.ActorRoutes(um, routes.ResourceNames{Actor: "members"})
	assertPatterns(t, rts, []string{
		"POST /members",
		"GET /members",
		"GET /members/{actorID}",
		"PATCH /members/{actorID}/display-name",
	})
}

func TestActorGroupAssignmentRoutes_DefaultResourceIsAssignments(t *testing.T) {
	um := newTestManager(t)
	rts := routes.ActorGroupAssignmentRoutes(um, routes.ResourceNames{})
	assertPatterns(t, rts, []string{
		"POST /actors/{actorID}/assignments",
		"DELETE /actors/{actorID}/assignments/{groupID}",
		"GET /actors/{actorID}/assignments",
		"GET /groups/{groupID}/assignments",
		"GET /assignments/home-counts",
	})
}

func TestActorGroupAssignmentRoutes_HonoursGroupOverride(t *testing.T) {
	um := newTestManager(t)
	rts := routes.ActorGroupAssignmentRoutes(um, routes.ResourceNames{Group: "chapters", Assignment: "registrations"})
	assertPatterns(t, rts, []string{
		"POST /actors/{actorID}/registrations",
		"DELETE /actors/{actorID}/registrations/{groupID}",
		"GET /actors/{actorID}/registrations",
		"GET /chapters/{groupID}/registrations",
		"GET /registrations/home-counts",
	})
}

func TestRoutes_ConcatenatesAllFourAndSharesNames(t *testing.T) {
	um := newTestManager(t)
	all := routes.Routes(um, stubHierarchy{}, routes.ResourceNames{Group: "chapters"})

	want := 3 + 4 + 5 + 4 // GroupRoutes + ActorRoutes + ActorGroupAssignmentRoutes + RoleRoutes
	if len(all) != want {
		t.Fatalf("got %d routes, want %d", len(all), want)
	}
	// The Group override must be visible in ActorGroupAssignmentRoutes' path too —
	// this is the whole point of sharing one ResourceNames.
	found := false
	for _, rt := range all {
		if rt.Pattern("") == "GET /chapters/{groupID}/assignments" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected /chapters/{groupID}/assignments among: %v", patterns(all, ""))
	}
}

func TestRoute_PatternWithPrefix(t *testing.T) {
	um := newTestManager(t)
	rts := routes.GroupRoutes(um, stubHierarchy{}, routes.ResourceNames{Group: "chapters"})

	if got := rts[0].Pattern("/v1/member"); got != "POST /v1/member/chapters" {
		t.Fatalf("got %q", got)
	}
}
