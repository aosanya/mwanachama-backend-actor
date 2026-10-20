package mwanachamaactor_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func newTestManagerWith(t *testing.T, opts ...mwanachamaactor.Option) (mwanachamaactor.UserManager, error) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamaactor.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := mwanachamaactor.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	return mwanachamaactor.NewUserManager(db, s, opts...)
}

func lifetimeOf(t *testing.T, inv mwanachamaactor.GroupInvitation) time.Duration {
	t.Helper()
	issued, err := time.Parse(time.RFC3339Nano, inv.InvitedAt)
	if err != nil {
		t.Fatalf("invited_at %q: %v", inv.InvitedAt, err)
	}
	expires, err := time.Parse(time.RFC3339Nano, inv.ExpiresAt)
	if err != nil {
		t.Fatalf("expires_at %q: %v", inv.ExpiresAt, err)
	}
	return expires.Sub(issued)
}

func TestAnInvitationWithNoDateLapsesAfterFourteenDaysByDefault(t *testing.T) {
	mgr := newTestManager(t)
	g, a := seedHidden(t, mgr)

	inv, err := mgr.InviteToGroup(context.Background(), g.ID, a.ID, "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	if mwanachamaactor.DefaultInvitationLifetime != 14*24*time.Hour {
		t.Errorf("DefaultInvitationLifetime = %s, want 14 days", mwanachamaactor.DefaultInvitationLifetime)
	}
	if got := lifetimeOf(t, inv); got != 14*24*time.Hour {
		t.Errorf("lifetime = %s, want 336h", got)
	}
	open, err := mgr.ListInvitationsForGroup(context.Background(), g.ID, true)
	if err != nil {
		t.Fatalf("ListInvitationsForGroup: %v", err)
	}
	if len(open) != 1 {
		t.Errorf("open invitations = %d, want 1", len(open))
	}
}

func TestTheHostSetsTheInvitationLifetime(t *testing.T) {
	mgr, err := newTestManagerWith(t, mwanachamaactor.WithInvitationLifetime(72*time.Hour))
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	g, a := seedHidden(t, mgr)

	inv, err := mgr.InviteToGroup(context.Background(), g.ID, a.ID, "leader-1", "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	if got := lifetimeOf(t, inv); got != 72*time.Hour {
		t.Errorf("lifetime = %s, want 72h", got)
	}
	stored, err := mgr.GetInvitation(context.Background(), inv.ID)
	if err != nil {
		t.Fatalf("GetInvitation: %v", err)
	}
	if stored.ExpiresAt != inv.ExpiresAt {
		t.Errorf("stored expires_at = %q, want %q", stored.ExpiresAt, inv.ExpiresAt)
	}
}

func TestAnInvitationWithItsOwnDateKeepsIt(t *testing.T) {
	mgr, err := newTestManagerWith(t, mwanachamaactor.WithInvitationLifetime(time.Hour))
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	g, a := seedHidden(t, mgr)
	own := time.Now().Add(30 * 24 * time.Hour).UTC()

	inv, err := mgr.InviteToGroup(context.Background(), g.ID, a.ID, "leader-1", own.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	if want := own.Format(mwanachamaactor.TimeLayout); inv.ExpiresAt != want {
		t.Errorf("expires_at = %q, want the issuer's %q", inv.ExpiresAt, want)
	}
}

func TestRenewingAnInvitationWithNoDateRestartsTheLifetime(t *testing.T) {
	mgr, err := newTestManagerWith(t, mwanachamaactor.WithInvitationLifetime(48*time.Hour))
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	g, a := seedHidden(t, mgr)
	ctx := context.Background()
	first, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-1", time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}

	again, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "leader-2", "")
	if err != nil {
		t.Fatalf("InviteToGroup again: %v", err)
	}
	if again.ID != first.ID {
		t.Fatalf("renewal id = %s, want the same row %s", again.ID, first.ID)
	}
	if got := lifetimeOf(t, again); got != 48*time.Hour {
		t.Errorf("renewed lifetime = %s, want 48h", got)
	}
}

func TestANonPositiveInvitationLifetimeIsRefused(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Hour} {
		if _, err := newTestManagerWith(t, mwanachamaactor.WithInvitationLifetime(d)); err == nil {
			t.Errorf("lifetime %s: NewUserManager succeeded, want an error", d)
		}
	}
}
