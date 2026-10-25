package mwanachamaactor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestAMemberCodeReadsAsThatPersonsScreenNameOnly(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina", FirstName: "Amina", LastName: "Odhiambo"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	got, err := mgr.ScreenNameByCode(ctx, "  "+strings.ToLower(a.PublicCode)+" ")
	if err != nil {
		t.Fatalf("ScreenNameByCode: %v", err)
	}
	if got.DisplayName != "Amina" {
		t.Errorf("screen name = %q, want Amina", got.DisplayName)
	}
	for _, code := range []string{"", "M-AAA-AAA-AAA", a.Code} {
		if _, err := mgr.ScreenNameByCode(ctx, code); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
			t.Errorf("ScreenNameByCode(%q) err = %v, want ErrActorNotFound", code, err)
		}
	}
}
