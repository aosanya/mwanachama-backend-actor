package models_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestAnInvitationWithNoInviterLeavesInvitedByOut(t *testing.T) {
	raw, err := json.Marshal(models.GroupInvitation{ID: "i-1", GroupID: "g-1", ActorID: "a-1", State: models.InvitationPending})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "invited_by") {
		t.Errorf("invitation = %s, want no invited_by key", raw)
	}
	raw, _ = json.Marshal(models.GroupInvitation{ID: "i-1", InvitedBy: "leader-1"})
	if !strings.Contains(string(raw), `"invited_by":"leader-1"`) {
		t.Errorf("invitation = %s, want the inviter kept when set", raw)
	}
}
