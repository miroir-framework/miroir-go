package jzodgen_test

import (
	"testing"

	"github.com/miroir-framework/miroir/go/miroirtest"
)

const jzodToGoTypeSuiteUUID = "1cfbb818-54d3-4429-b934-477ae020f374"

func TestJzodToGoType(t *testing.T) {
	outcomes, err := miroirtest.RunFile(miroirtest.TestsFile(jzodToGoTypeSuiteUUID))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 48 {
		t.Fatalf("leaves: got %d want 48", len(outcomes))
	}
	for _, o := range outcomes {
		if !o.OK {
			t.Errorf("%s: %s", o.Label, o.Err)
		}
	}
}
