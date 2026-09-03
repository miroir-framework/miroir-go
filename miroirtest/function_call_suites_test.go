package miroirtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnitFunctionCallSuites(t *testing.T) {
	runFunctionCallJSONDir(t, MiroirTestDir())
	runFunctionCallJSONDir(t, TestsDir())
}

func runFunctionCallJSONDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		doc, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		leaves := WalkLeaves(doc)
		var fc []map[string]any
		for _, leaf := range leaves {
			if leaf["miroirTestType"] == "functionCallTest" {
				fc = append(fc, leaf)
			}
		}
		if len(fc) == 0 {
			continue
		}
		name, _ := doc["name"].(string)
		t.Run(name, func(t *testing.T) {
			for _, leaf := range fc {
				o := runFunctionCall(fmtLabel(leaf), leaf)
				if !o.OK {
					t.Errorf("%s: %s", o.Label, o.Err)
				}
			}
		})
	}
}

func fmtLabel(leaf map[string]any) string {
	label, _ := leaf["miroirTestLabel"].(string)
	return label
}
