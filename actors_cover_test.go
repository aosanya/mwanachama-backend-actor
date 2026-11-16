package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestEnsureActorCreatesOnceAndThenReturnsTheSameActor(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	if _, err := um.EnsureActor(ctx, "  "); !errors.Is(err, actor.ErrInvalidActor) {
		t.Errorf("blank id: %v", err)
	}
	first, err := um.EnsureActor(ctx, " member-17 ")
	if err != nil || first.ID != "member-17" || first.PublicCode == "" || !first.FormsWeekSplit {
		t.Fatalf("EnsureActor = %+v %v", first, err)
	}
	second, err := um.EnsureActor(ctx, "member-17")
	if err != nil || second.ID != first.ID || second.Code != first.Code {
		t.Errorf("second EnsureActor = %+v %v, want the first actor back", second, err)
	}
	all, _ := um.ListActors(ctx)
	if len(all) != 1 {
		t.Errorf("actors = %d, want 1", len(all))
	}
}

func TestActorsAreReadManyAtATimeWithoutDuplicatesOrMissingOnes(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Árva Ádám"})
	b, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Bakó Bálint"})
	got, err := um.GetActors(ctx, []string{b.ID, a.ID, b.ID, "ghost"})
	if err != nil {
		t.Fatalf("GetActors: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d actors, want the 2 that exist once each", len(got))
	}
	if got[0].ID > got[1].ID {
		t.Errorf("actors are not ordered by id: %q then %q", got[0].ID, got[1].ID)
	}
	none, err := um.GetActors(ctx, nil)
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("GetActors(nil) = %v %v, want an empty list", none, err)
	}
}

func TestAnActorIsFoundByPhoneAndAPhoneIsUnique(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Vég Vilma", Attributes: map[string]any{"phone": "+3612345678"}})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	found, err := um.GetActorByPhone(ctx, "+3612345678")
	if err != nil || found.ID != a.ID {
		t.Fatalf("GetActorByPhone = %+v %v", found, err)
	}
	if _, err := um.GetActorByPhone(ctx, "+3600000000"); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown phone: %v", err)
	}
	if _, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Másik Mari", Attributes: map[string]any{"phone": "+3612345678"}}); !errors.Is(err, actor.ErrDuplicateAttribute) {
		t.Errorf("the same phone twice: %v", err)
	}
	if _, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Rossz Rita", Attributes: map[string]any{"phone": 12345}}); !errors.Is(err, actor.ErrInvalidActor) {
		t.Errorf("a phone that is not text: %v", err)
	}
	kept, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Fax Feri", Attributes: map[string]any{"fax": "1"}})
	if err != nil || kept.Attributes["fax"] != "1" {
		t.Errorf("an undeclared attribute is not a closed-schema error on create: %+v %v", kept, err)
	}
}

func TestRenamingAnActorChangesOnlyTheNames(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Régi Név"})
	shown, err := um.SetActorDisplayName(ctx, a.ID, "Új Név")
	if err != nil || shown.DisplayName != "Új Név" {
		t.Fatalf("SetActorDisplayName = %+v %v", shown, err)
	}
	named, err := um.SetActorName(ctx, a.ID, "Új", "Név")
	if err != nil || named.FirstName != "Új" || named.LastName != "Név" || named.DisplayName != "Új Név" {
		t.Fatalf("SetActorName = %+v %v", named, err)
	}
	read, _ := um.GetActor(ctx, a.ID)
	if read.DisplayName != "Új Név" || read.FirstName != "Új" || read.LastName != "Név" || read.Code != a.Code {
		t.Errorf("stored = %+v", read)
	}
	if _, err := um.SetActorDisplayName(ctx, "nobody", "X"); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown actor display name: %v", err)
	}
	if _, err := um.SetActorName(ctx, "nobody", "X", "Y"); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown actor name: %v", err)
	}
}

func TestAPersonCorrectsTheirOwnAttributesAndTheSourceIsRecorded(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Javító Jenő"})
	fixed, err := um.CorrectMyAttributes(ctx, a.ID, models.AttributeCorrection{Attributes: map[string]any{"email": "jeno@example.org"}})
	if err != nil {
		t.Fatalf("CorrectMyAttributes: %v", err)
	}
	src := fixed.AttributeSources["email"]
	if fixed.Attributes["email"] != "jeno@example.org" || src.SetBy != models.AttributeSetBySelf || src.SetByID != a.ID || src.SetAt == "" {
		t.Errorf("fixed = %+v", fixed)
	}
	staff, err := um.SetActorAttributes(ctx, a.ID, "staff-1", models.AttributeCorrection{Attributes: map[string]any{"phone": "+3611111111"}})
	if err != nil {
		t.Fatalf("SetActorAttributes: %v", err)
	}
	if staff.AttributeSources["phone"].SetBy != models.AttributeSetByStaff || staff.AttributeSources["phone"].SetByID != "staff-1" {
		t.Errorf("staff source = %+v", staff.AttributeSources["phone"])
	}
	if staff.AttributeSources["email"].SetBy != models.AttributeSetBySelf || staff.Attributes["email"] != "jeno@example.org" {
		t.Errorf("an earlier attribute was disturbed: %+v", staff)
	}
	cleared, err := um.CorrectMyAttributes(ctx, a.ID, models.AttributeCorrection{Attributes: map[string]any{"email": ""}})
	if err != nil {
		t.Fatalf("clearing: %v", err)
	}
	if _, still := cleared.Attributes["email"]; still {
		t.Errorf("an emptied attribute is still held: %+v", cleared.Attributes)
	}
	byPhone, err := um.GetActorByPhone(ctx, "+3611111111")
	if err != nil || byPhone.ID != a.ID {
		t.Errorf("the corrected phone is not stored: %+v %v", byPhone, err)
	}
}

func TestAttributeCorrectionsAreRefusedWhenTheyMakeNoSense(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Egy Ede", Attributes: map[string]any{"phone": "+3622222222"}})
	b, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Két Kata"})
	if _, err := um.CorrectMyAttributes(ctx, " ", models.AttributeCorrection{}); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("no actor: %v", err)
	}
	if _, err := um.CorrectMyAttributes(ctx, "nobody", models.AttributeCorrection{Attributes: map[string]any{"email": "x@example.org"}}); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown actor: %v", err)
	}
	if _, err := um.CorrectMyAttributes(ctx, b.ID, models.AttributeCorrection{}); !errors.Is(err, actor.ErrInvalidActor) {
		t.Errorf("nothing to set: %v", err)
	}
	if _, err := um.CorrectMyAttributes(ctx, b.ID, models.AttributeCorrection{Attributes: map[string]any{"shoe_size": 42}}); !errors.Is(err, actor.ErrInvalidActor) {
		t.Errorf("undeclared attribute: %v", err)
	}
	if _, err := um.CorrectMyAttributes(ctx, b.ID, models.AttributeCorrection{Attributes: map[string]any{"phone": "+3622222222"}}); !errors.Is(err, actor.ErrDuplicateAttribute) {
		t.Errorf("a phone someone else holds: %v", err)
	}
	if _, err := um.SetActorAttributes(ctx, a.ID, " ", models.AttributeCorrection{Attributes: map[string]any{"email": "x@example.org"}}); !errors.Is(err, actor.ErrInvalidActor) {
		t.Errorf("a staff edit without an editor: %v", err)
	}
	again, err := um.CorrectMyAttributes(ctx, a.ID, models.AttributeCorrection{Attributes: map[string]any{"phone": "+3622222222"}})
	if err != nil || again.Attributes["phone"] != "+3622222222" {
		t.Errorf("setting your own phone to what you already hold: %+v %v", again, err)
	}
}

func TestFormsWeekSplitAndTheHomeCounts(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	a, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Heti Hanna"})
	pref, err := um.GetMyFormsWeekSplit(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetMyFormsWeekSplit: %v", err)
	}
	if !pref.FormsWeekSplit {
		t.Errorf("default preference = %+v, want the week split on", pref)
	}
	set, err := um.SetMyFormsWeekSplit(ctx, a.ID, false)
	if err != nil || set.FormsWeekSplit {
		t.Fatalf("SetMyFormsWeekSplit = %+v %v", set, err)
	}
	read, _ := um.GetMyFormsWeekSplit(ctx, a.ID)
	if read.FormsWeekSplit {
		t.Errorf("stored preference = %+v, want off", read)
	}
	unknown, err := um.GetMyFormsWeekSplit(ctx, "nobody")
	if err != nil || !unknown.FormsWeekSplit || unknown.ActorID != "nobody" {
		t.Errorf("an unknown actor reads the default: %+v %v", unknown, err)
	}
	blank, err := um.GetMyFormsWeekSplit(ctx, " ")
	if err != nil || !blank.FormsWeekSplit {
		t.Errorf("no actor reads the default: %+v %v", blank, err)
	}
	if _, err := um.SetMyFormsWeekSplit(ctx, " ", true); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("blank actor set: %v", err)
	}
	if _, err := um.SetMyFormsWeekSplit(ctx, "nobody", true); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown actor set: %v", err)
	}
}
