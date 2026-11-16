package routes_test

import (
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/routes"
)

func TestTheApprovalLimitRoutesAndToolsAreDeclared(t *testing.T) {
	declared := map[string]bool{}
	for _, rt := range routes.Shape() {
		declared[rt.Method+" "+rt.Path] = true
	}
	for _, want := range []string{
		"PUT /role-assignments/{assignment_id}/approval-limit",
		"POST /role-assignments/{assignment_id}/authorise-approval",
	} {
		if !declared[want] {
			t.Errorf("the declared table has no route %s", want)
		}
	}
	tools, err := routes.BuildTools(newManager(t))
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	published := map[string]bool{}
	for _, tool := range tools {
		published[tool.Name] = true
	}
	for _, want := range []string{"actor_set_approval_limit", "actor_authorise_approval"} {
		if !published[want] {
			t.Errorf("the declared table publishes no tool %s", want)
		}
	}
}
