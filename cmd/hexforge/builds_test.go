package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/forge"
)

// TestTheBuildListingNamesEverythingABuildIs.
//
// The catalogue is the one file here whose whole purpose is to be read by
// somebody choosing between directions, and until this command existed the only
// way to read it was to open the JSON. So what is asserted is that every field a
// build carries reaches the page: an id, a name, whose it is, the FORM it is
// fielded as, the trait and the four skills.
//
// The trait is checked as carefully as the kit for the reason
// TestTheShippedBuildsAreTheOnesTheTestsMeasure compares it: a kit is half a
// build, and two directions can differ by the trait alone.
func TestTheBuildListingNamesEverythingABuildIs(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	builds := lib.Builds()
	if len(builds) == 0 {
		t.Fatal("no builds are shipped, so this listing has nothing to draw")
	}
	var page bytes.Buffer
	renderBuilds(&page, lib)
	drawn := page.String()

	for _, build := range builds {
		for what, text := range map[string]string{
			"id":        build.ID,
			"name":      build.Name,
			"character": build.Character,
			"form":      build.Stage,
			"trait":     build.Passives[0],
			"intent":    build.Intent,
		} {
			if !strings.Contains(drawn, text) {
				t.Errorf("the listing never says the %s of %s (%q)", what, build.ID, text)
			}
		}
		for _, skill := range build.Skills {
			if !strings.Contains(drawn, skill) {
				t.Errorf("the listing never says %s, which %s brings", skill, build.ID)
			}
		}
	}
}

// TestTheBuildListingSaysWhoHasNoChoice is the count under the table, and it is
// the fact the catalogue exists to report.
//
// A character with two directions has a decision; one with none is the honest
// case that TestABuildIsACatalogueOfChoicesRatherThanOfKits protects, and a
// listing that only counted builds would let a reader assume the whole cast was
// covered.
func TestTheBuildListingSaysWhoHasNoChoice(t *testing.T) {
	lib, err := forge.Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load the shipped data: %v", err)
	}
	withBuilds, without := 0, []string(nil)
	for _, character := range lib.Characters().All() {
		if len(lib.BuildsOf(character.ID)) > 0 {
			withBuilds++
			continue
		}
		without = append(without, character.ID)
	}
	if len(without) == 0 {
		t.Skip("every shipped character has a build, so the count below cannot be told from a head count")
	}
	var page bytes.Buffer
	renderBuilds(&page, lib)
	if got := page.String(); !strings.Contains(got, "across") {
		t.Errorf("the listing does not say how much of the cast it covers:\n%s", got)
	}
	t.Logf("%d of %d characters have a direction; %v have none",
		withBuilds, len(lib.Characters().All()), without)
}

// TestAnEmptyCatalogueSaysSoRatherThanDrawingAnEmptyTable.
//
// A header row with nothing under it reads as a listing that failed, which is
// the same judgement every other listing here already makes.
func TestAnEmptyCatalogueSaysSoRatherThanDrawingAnEmptyTable(t *testing.T) {
	dir := t.TempDir()
	copyTree(t, shippedDataDir, dir)
	if err := os.WriteFile(filepath.Join(dir, "builds.json"),
		[]byte(`{"builds":[]}`), 0o600); err != nil {
		t.Fatalf("empty the catalogue: %v", err)
	}
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load the emptied data: %v", err)
	}
	if len(lib.Builds()) != 0 {
		t.Fatalf("the catalogue still holds %d builds", len(lib.Builds()))
	}
	var page bytes.Buffer
	renderBuilds(&page, lib)
	got := page.String()
	if strings.Contains(got, "id") && strings.Contains(got, "intent") {
		t.Errorf("an empty catalogue drew its header row:\n%s", got)
	}
	if !strings.Contains(got, "no builds") {
		t.Errorf("an empty catalogue does not say so:\n%s", got)
	}
}
