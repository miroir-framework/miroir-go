package miroirtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnitTransformerSuites(t *testing.T) {
	entries, err := os.ReadDir(MiroirTestDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(MiroirTestDir(), e.Name())
		doc, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var tt []map[string]any
		for _, leaf := range WalkLeaves(doc) {
			if leaf["miroirTestType"] == "transformerTest" {
				tt = append(tt, leaf)
			}
		}
		if len(tt) == 0 {
			continue
		}
		name, _ := doc["name"].(string)
		t.Run(name, func(t *testing.T) {
			for i, leaf := range tt {
				label := fmtLabel(leaf)
				o := runTransformerTest(label, leaf)
				if !o.OK {
					t.Errorf("[%d] %s: %s", i, o.Label, o.Err)
				}
			}
		})
	}
}
