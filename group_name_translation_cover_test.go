package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
)

func TestTranslationRefusesABadLanguageTagABlankNameOrAnUnknownGroup(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Északi kerület", "")
	for _, language := range []string{"", "english", "E", "en_US", "en-"} {
		if _, err := um.SetGroupNameTranslation(ctx, g.ID, language, "North"); !errors.Is(err, actor.ErrInvalidLanguage) {
			t.Errorf("language %q: err = %v, want ErrInvalidLanguage", language, err)
		}
	}
	if _, err := um.SetGroupNameTranslation(ctx, g.ID, "en", "  "); !errors.Is(err, actor.ErrInvalidTranslation) {
		t.Errorf("blank name: err = %v", err)
	}
	if _, err := um.SetGroupNameTranslation(ctx, "nowhere", "en", "North"); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: err = %v", err)
	}
}

func TestTranslationIsStoredInLowerCaseAndNeedsAReviewerBeforeItIsUsed(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Északi kerület", "")
	set, err := um.SetGroupNameTranslation(ctx, g.ID, " PT-BR ", "  Distrito Norte ")
	if err != nil {
		t.Fatalf("SetGroupNameTranslation: %v", err)
	}
	if set.Language != "pt-br" || set.Name != "Distrito Norte" || set.Reviewed {
		t.Fatalf("translation = %+v", set)
	}
	names, err := um.GroupNamesInLanguage(ctx, "pt-br", []string{g.ID})
	if err != nil || len(names) != 0 {
		t.Fatalf("names before review = %v, %v; want none", names, err)
	}
	if _, err := um.ReviewGroupNameTranslation(ctx, g.ID, "pt-br", " "); !errors.Is(err, actor.ErrReviewerRequired) {
		t.Errorf("review by nobody: err = %v", err)
	}
	reviewed, err := um.ReviewGroupNameTranslation(ctx, g.ID, "pt-br", "ana")
	if err != nil || !reviewed.Reviewed || reviewed.ReviewedBy != "ana" || reviewed.ReviewedAt == "" {
		t.Fatalf("reviewed = %+v, %v", reviewed, err)
	}
	names, _ = um.GroupNamesInLanguage(ctx, "PT-BR", []string{g.ID})
	if names[g.ID] != "Distrito Norte" {
		t.Fatalf("names after review = %v", names)
	}
}

func TestEditingAReviewedTranslationClearsTheReviewButTheSameNameKeepsIt(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Északi kerület", "")
	um.SetGroupNameTranslation(ctx, g.ID, "de", "Nordbezirk")
	um.ReviewGroupNameTranslation(ctx, g.ID, "de", "ana")
	same, err := um.SetGroupNameTranslation(ctx, g.ID, "de", "Nordbezirk")
	if err != nil || !same.Reviewed {
		t.Fatalf("setting the same name = %+v, %v; the review must stay", same, err)
	}
	edited, err := um.SetGroupNameTranslation(ctx, g.ID, "de", "Nördlicher Bezirk")
	if err != nil || edited.Reviewed || edited.ReviewedBy != "" || edited.ReviewedAt != "" || edited.Name != "Nördlicher Bezirk" {
		t.Fatalf("edited = %+v, %v; the review must be cleared", edited, err)
	}
	names, _ := um.GroupNamesInLanguage(ctx, "de", []string{g.ID})
	if len(names) != 0 {
		t.Fatalf("an unreviewed edit is already used: %v", names)
	}
}

func TestReviewRefusesAMissingTranslationOrLanguage(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Északi kerület", "")
	if _, err := um.ReviewGroupNameTranslation(ctx, g.ID, "fr", "ana"); !errors.Is(err, actor.ErrTranslationNotFound) {
		t.Errorf("nothing set: err = %v", err)
	}
	if _, err := um.ReviewGroupNameTranslation(ctx, g.ID, "??", "ana"); !errors.Is(err, actor.ErrInvalidLanguage) {
		t.Errorf("bad language: err = %v", err)
	}
}

func TestTranslationsListByLanguageAndOnlyForAKnownGroup(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Északi kerület", "")
	um.SetGroupNameTranslation(ctx, g.ID, "sw", "Wilaya ya Kaskazini")
	um.SetGroupNameTranslation(ctx, g.ID, "de", "Nordbezirk")
	got, err := um.ListGroupNameTranslations(ctx, g.ID)
	if err != nil || len(got) != 2 || got[0].Language != "de" || got[1].Language != "sw" {
		t.Fatalf("translations = %+v, %v; want de then sw", got, err)
	}
	if _, err := um.ListGroupNameTranslations(ctx, "nowhere"); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: err = %v", err)
	}
}

func TestGroupInALanguageUsesTheReviewedNameAndFallsBackToItsOwn(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	g := groupNamed(t, um, "Északi kerület", "")
	um.SetGroupNameTranslation(ctx, g.ID, "de", "Nordbezirk")
	um.ReviewGroupNameTranslation(ctx, g.ID, "de", "ana")
	german, err := um.GetGroupInLanguage(ctx, g.ID, "de")
	if err != nil || german.Name != "Nordbezirk" {
		t.Fatalf("in German = %+v, %v", german, err)
	}
	french, err := um.GetGroupInLanguage(ctx, g.ID, "fr")
	if err != nil || french.Name != "Északi kerület" {
		t.Fatalf("in French = %+v, %v; want its own name", french, err)
	}
	own, err := um.GetGroupInLanguage(ctx, g.ID, "")
	if err != nil || own.Name != "Északi kerület" {
		t.Fatalf("with no language = %+v, %v", own, err)
	}
	if _, err := um.GetGroupInLanguage(ctx, "nowhere", "de"); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: err = %v", err)
	}
}

func TestGroupNamesInALanguageAsksNothingForNoLanguageOrNoGroups(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	if names, err := um.GroupNamesInLanguage(ctx, " ", []string{"a"}); err != nil || len(names) != 0 {
		t.Errorf("blank language = %v, %v", names, err)
	}
	if names, err := um.GroupNamesInLanguage(ctx, "de", nil); err != nil || len(names) != 0 {
		t.Errorf("no groups = %v, %v", names, err)
	}
	if _, err := um.GroupNamesInLanguage(ctx, "not a tag", []string{"a"}); !errors.Is(err, actor.ErrInvalidLanguage) {
		t.Errorf("bad language: err = %v", err)
	}
}
