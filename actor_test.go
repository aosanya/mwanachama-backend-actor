package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestCreateActor_MintsIDAndCreatedAt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		DisplayName: "Amina",
		Attributes:  map[string]any{"email": "amina@example.com"},
	})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if a.ID == "" {
		t.Error("expected a minted ID")
	}
	if a.CreatedAt == "" {
		t.Error("expected CreatedAt to be stamped")
	}
	if a.DisplayName != "Amina" || a.Attributes["email"] != "amina@example.com" {
		t.Errorf("round-trip mismatch: %+v", a)
	}
	if a.IsAgentic {
		t.Error("IsAgentic should default false")
	}
	if a.Code != "AC-1" {
		t.Errorf("Code = %q, want AC-1", a.Code)
	}
}

func TestCreateActor_CodeIsSequentialAndImmutable(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	a1, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Amina"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	a2, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if a1.Code != "AC-1" || a2.Code != "AC-2" {
		t.Fatalf("Codes = %q, %q, want AC-1, AC-2 (sequential across repeated Creates)", a1.Code, a2.Code)
	}

	// Code survives SetActorDisplayName unchanged — that call writes
	// display_name/updated_at only, never code.
	renamed, err := mgr.SetActorDisplayName(ctx, a1.ID, "Amina Renamed")
	if err != nil {
		t.Fatalf("SetActorDisplayName: %v", err)
	}
	if renamed.Code != a1.Code {
		t.Fatalf("Code changed after SetActorDisplayName: got %q, want %q", renamed.Code, a1.Code)
	}
}

func TestCreateActor_WithAttributes_RoundTrips(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		DisplayName: "Agentic One",
		IsAgentic:   true,
		Attributes:  map[string]any{"persona": "farmer"},
	})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	got, err := mgr.GetActor(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetActor: %v", err)
	}
	if !got.IsAgentic {
		t.Error("expected IsAgentic to round-trip true")
	}
	if got.Attributes["persona"] != "farmer" {
		t.Errorf("Attributes = %v, want persona=farmer", got.Attributes)
	}
}

func TestCreateActor_RejectsWrongRangeAttribute(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		DisplayName: "Bad Phone",
		Attributes:  map[string]any{"phone": 254712345678}, // phone is RangeText, not a number
	})
	if !errors.Is(err, mwanachamaactor.ErrInvalidActor) {
		t.Fatalf("err = %v, want ErrInvalidActor", err)
	}
}

func TestCreateActor_RejectsDuplicateUniqueAttribute(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	if _, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		DisplayName: "First",
		Attributes:  map[string]any{"phone": "254712345678"},
	}); err != nil {
		t.Fatalf("CreateActor(first): %v", err)
	}

	_, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{
		DisplayName: "Second",
		Attributes:  map[string]any{"phone": "254712345678"},
	})
	if !errors.Is(err, mwanachamaactor.ErrDuplicateAttribute) {
		t.Fatalf("err = %v, want ErrDuplicateAttribute", err)
	}
}

func TestCreateActor_NoPhoneOrEmailIsNotRequired(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// Mirrors the gateway's registerDevice, which mints a member before any
	// phone number is given, and agentic actors, which are forbidden one.
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "No Contact Info"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	if a.Attributes["phone"] != nil || a.Attributes["email"] != nil {
		t.Errorf("expected no phone/email, got %+v", a.Attributes)
	}
}

func TestGetActor_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.GetActor(context.Background(), "nope"); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Fatalf("GetActor err = %v, want ErrActorNotFound", err)
	}
}

func TestGetActors_SkipsMissing_SortsByID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	a, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "A"})
	b, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "B"})

	out, err := mgr.GetActors(ctx, []string{b.ID, "missing", a.ID, a.ID})
	if err != nil {
		t.Fatalf("GetActors: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	// sorted by id
	if !(out[0].ID < out[1].ID) {
		t.Errorf("expected sorted by id, got %v then %v", out[0].ID, out[1].ID)
	}
}

func TestSetActorDisplayName(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	a, _ := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Old"})
	updated, err := mgr.SetActorDisplayName(ctx, a.ID, "New")
	if err != nil {
		t.Fatalf("SetActorDisplayName: %v", err)
	}
	if updated.DisplayName != "New" {
		t.Errorf("DisplayName = %q, want New", updated.DisplayName)
	}
	if updated.CreatedAt != a.CreatedAt {
		t.Errorf("CreatedAt changed: %q -> %q", a.CreatedAt, updated.CreatedAt)
	}
}

func TestSetActorDisplayName_NotFound(t *testing.T) {
	mgr := newTestManager(t)
	if _, err := mgr.SetActorDisplayName(context.Background(), "nope", "x"); !errors.Is(err, mwanachamaactor.ErrActorNotFound) {
		t.Fatalf("err = %v, want ErrActorNotFound", err)
	}
}

func TestListActors_SortedByID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	for _, name := range []string{"Zeta", "Alpha", "Mid"} {
		if _, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: name}); err != nil {
			t.Fatalf("CreateActor: %v", err)
		}
	}
	out, err := mgr.ListActors(ctx)
	if err != nil {
		t.Fatalf("ListActors: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	for i := 1; i < len(out); i++ {
		if out[i-1].ID > out[i].ID {
			t.Fatalf("ListActors not sorted by id: %v", out)
		}
	}
}
