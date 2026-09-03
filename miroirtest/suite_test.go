package miroirtest

import "testing"

func TestRunMustacheSuite(t *testing.T) {
	outcomes, err := RunFile(MiroirTestFile("bdf83d4d-f4dd-42c9-b2d6-41311d979083"))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 6 {
		t.Fatalf("outcomes: got %d", len(outcomes))
	}
	for _, o := range outcomes {
		if !o.OK {
			t.Errorf("%s: %s", o.Label, o.Err)
		}
	}
}

func TestRunAlterObjectSuite(t *testing.T) {
	outcomes, err := RunFile(MiroirTestFile("d3b7f54f-8dcf-4159-814e-0f4a71a6081a"))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 3 {
		t.Fatalf("outcomes: got %d", len(outcomes))
	}
	for _, o := range outcomes {
		if !o.OK {
			t.Errorf("%s: %s", o.Label, o.Err)
		}
	}
}

func TestFailClosedQueryTest(t *testing.T) {
	leaf := map[string]any{
		"miroirTestType":  "queryTest",
		"miroirTestLabel": "not-yet",
	}
	got := runLeaf(leaf)
	if got.OK {
		t.Fatal("queryTest must fail closed")
	}
}

func TestTracerReturnValueEmptyArray(t *testing.T) {
	outcomes, err := RunFile(MiroirTestFile("33f60ac8-6511-43b1-b153-6b86e3177532"))
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) == 0 {
		t.Fatal("no transformer leaves")
	}
	o := outcomes[0]
	if !o.OK {
		t.Fatalf("tracer leaf %s: %s", o.Label, o.Err)
	}
}
