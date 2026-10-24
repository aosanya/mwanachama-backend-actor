package models

import "testing"

func TestFoldStreetNameIgnoresCaseAccentsAndPunctuation(t *testing.T) {
	cases := map[string]string{
		"Petőfi Sándor utca":      "petofi sandor utca",
		"  PETŐFI   sándor UTCA ": "petofi sandor utca",
		"Petofi Sandor u.":        "petofi sandor u",
		"Árpád-fejedelem útja":    "arpad fejedelem utja",
		"Straße 12":               "straße 12",
		"":                        "",
		" .,- ":                   "",
	}
	for in, want := range cases {
		if got := FoldStreetName(in); got != want {
			t.Errorf("FoldStreetName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldStreetNameLeavesAbbreviationsUnexpanded(t *testing.T) {
	if FoldStreetName("Kossuth u.") == FoldStreetName("Kossuth utca") {
		t.Fatal("abbreviation expansion is not decided; folding must not expand u. to utca")
	}
}
