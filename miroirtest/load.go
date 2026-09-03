package miroirtest

import (
	"fmt"
	"os"

	"github.com/miroir-framework/miroir/go/jzod"
)

// LoadFile reads a MiroirTest JSON instance, preserving object key order via
// [jzod.Decode].
func LoadFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoded, err := jzod.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	doc, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unmarshal %s: root is not an object", path)
	}
	return doc, nil
}

// WalkLeaves returns every leaf under doc (or doc.definition) in document
// order: functionCallTest, transformerTest, queryTest, actionTest, runnerTest.
func WalkLeaves(doc map[string]any) []map[string]any {
	var out []map[string]any
	root := doc
	if def, ok := doc["definition"].(map[string]any); ok {
		if _, hasType := def["miroirTestType"]; hasType {
			root = def
		}
	}
	walkLeaves(root, &out)
	return out
}

func walkLeaves(node map[string]any, out *[]map[string]any) {
	switch node["miroirTestType"] {
	case "miroirTestSuite":
		tests, _ := node["miroirTests"].([]any)
		for _, child := range tests {
			if m, ok := child.(map[string]any); ok {
				walkLeaves(m, out)
			}
		}
	case "functionCallTest", "transformerTest", "queryTest", "actionTest", "runnerTest":
		*out = append(*out, node)
	}
}
