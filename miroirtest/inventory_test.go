package miroirtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryMiroirTestDirectory(t *testing.T) {
	entries, err := os.ReadDir(MiroirTestDir())
	if err != nil {
		t.Fatal(err)
	}
	jsonCount := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			jsonCount++
		}
	}
	if jsonCount != 51 {
		t.Fatalf("MiroirTest JSON files: got %d want 51", jsonCount)
	}
}

func TestInventoryMustacheSuite(t *testing.T) {
	doc, err := LoadFile(MiroirTestFile("bdf83d4d-f4dd-42c9-b2d6-41311d979083"))
	if err != nil {
		t.Fatal(err)
	}
	if doc["name"] != "mustache" {
		t.Fatalf("name: got %v", doc["name"])
	}
	leaves := WalkLeaves(doc)
	if len(leaves) != 6 {
		t.Fatalf("mustache leaves: got %d want 6", len(leaves))
	}
	for i, leaf := range leaves {
		if leaf["miroirTestType"] != "functionCallTest" {
			t.Fatalf("leaf %d type: %v", i, leaf["miroirTestType"])
		}
	}
	ref := asMap(leaves[0]["functionRef"])
	if ref["module"] != "miroir-core/1_core/mustache" || ref["export"] != "extractDoubleBracePatterns" {
		t.Fatalf("first functionRef: %#v", ref)
	}
}

func TestInventoryAlterObjectSuite(t *testing.T) {
	doc, err := LoadFile(MiroirTestFile("d3b7f54f-8dcf-4159-814e-0f4a71a6081a"))
	if err != nil {
		t.Fatal(err)
	}
	leaves := WalkLeaves(doc)
	if len(leaves) != 3 {
		t.Fatalf("alterObject leaves: got %d want 3", len(leaves))
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}
