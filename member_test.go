package mwanachamauser_test

import (
	"context"
	"errors"
	"testing"

	mwanachamauser "github.com/aosanya/mwanachama-backend-user"
)

func newTestManager(t *testing.T) mwanachamauser.UserManager {
	t.Helper()
	mgr, err := mwanachamauser.NewUserManager(newFakeDataManager())
	if err != nil {
		t.Fatalf("NewUserManager: %v", err)
	}
	return mgr
}

func TestNewUserManager_NilDataManager(t *testing.T) {
	if _, err := mwanachamauser.NewUserManager(nil); err == nil {
		t.Fatal("expected error for nil DataManager")
	}
}

func TestCreateMember_MintsIDAndCreatedAt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	m, err := mgr.CreateMember(ctx, mwanachamauser.Member{
		DisplayName: "Amina",
		Email:       "amina@example.com",
	})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}
	if m.ID == "" {
		t.Error("expected a minted ID")
	}
	if m.CreatedAt == "" {
		t.Error("expected CreatedAt to be stamped")
	}
	if m.DisplayName != "Amina" || m.Email != "amina@example.com" {
		t.Errorf("round-trip mismatch: %+v", m)
	}
	if m.IsAgentic {
		t.Error("IsAgentic should default false")
	}
}

func TestCreateMember_WithAttributes_RoundTrips(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	m, err := mgr.CreateMember(ctx, mwanachamauser.Member{
		DisplayName: "Agentic One",
		IsAgentic:   true,
		Attributes:  map[string]any{"persona": "farmer"},
	})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}
	got, err := mgr.GetMember(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if !got.IsAgentic {
		t.Error("expected IsAgentic to round-trip true")
	}
	if got.Attributes["persona"] != "farmer" {
		t.Errorf("Attributes = %v, want persona=farmer", got.Attributes)
	}
}

func TestGetMember_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetMember(context.Background(), "nope"); !errors.Is(err, mwanachamauser.ErrMemberNotFound) {
		t.Fatalf("GetMember err = %v, want ErrMemberNotFound", err)
	}
}

func TestGetMembers_SkipsMissing_SortsByID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	a, _ := mgr.CreateMember(ctx, mwanachamauser.Member{DisplayName: "A"})
	b, _ := mgr.CreateMember(ctx, mwanachamauser.Member{DisplayName: "B"})

	out, err := mgr.GetMembers(ctx, []string{b.ID, "missing", a.ID, a.ID})
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	// sorted by id
	if !(out[0].ID < out[1].ID) {
		t.Errorf("expected sorted by id, got %v then %v", out[0].ID, out[1].ID)
	}
}

func TestSetMemberDisplayName(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	m, _ := mgr.CreateMember(ctx, mwanachamauser.Member{DisplayName: "Old"})
	updated, err := mgr.SetMemberDisplayName(ctx, m.ID, "New")
	if err != nil {
		t.Fatalf("SetMemberDisplayName: %v", err)
	}
	if updated.DisplayName != "New" {
		t.Errorf("DisplayName = %q, want New", updated.DisplayName)
	}
	if updated.CreatedAt != m.CreatedAt {
		t.Errorf("CreatedAt changed: %q -> %q", m.CreatedAt, updated.CreatedAt)
	}
}

func TestSetMemberDisplayName_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.SetMemberDisplayName(context.Background(), "nope", "x"); !errors.Is(err, mwanachamauser.ErrMemberNotFound) {
		t.Fatalf("err = %v, want ErrMemberNotFound", err)
	}
}

func TestListMembers_SortedByID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	for _, name := range []string{"Zeta", "Alpha", "Mid"} {
		if _, err := mgr.CreateMember(ctx, mwanachamauser.Member{DisplayName: name}); err != nil {
			t.Fatalf("CreateMember: %v", err)
		}
	}
	out, err := mgr.ListMembers(ctx)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	for i := 1; i < len(out); i++ {
		if out[i-1].ID > out[i].ID {
			t.Fatalf("ListMembers not sorted by id: %v", out)
		}
	}
}
