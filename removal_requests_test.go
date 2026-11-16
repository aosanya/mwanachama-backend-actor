package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestRemovingAnActorEndsTheirPendingJoinRequestsSoALaterOperatorApprovalIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	res, err := mgr.JoinGroup(ctx, g.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	if !res.Request.Pending() {
		t.Fatalf("request = %+v, want pending", res.Request)
	}

	if _, err := mgr.RemoveFromAllGroups(ctx, a.ID, "operator-1"); err != nil {
		t.Fatalf("RemoveFromAllGroups: %v", err)
	}

	asked, err := mgr.ListJoinRequestsForActor(ctx, a.ID)
	if err != nil || len(asked) != 1 || asked[0].State != models.JoinRequestDeclined {
		t.Fatalf("requests = %+v, %v; want the one request declined", asked, err)
	}
	if _, err := mgr.OperatorApproveJoinRequest(ctx, res.Request.ID, "operator-1"); !errors.Is(err, mwanachamaactor.ErrJoinRequestDecided) {
		t.Errorf("approval after removal = %v, want ErrJoinRequestDecided", err)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Error("a removed actor was seated by a late approval")
	}
}
