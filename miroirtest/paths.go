package miroirtest

import (
	"path/filepath"
	"runtime"
)

func packageDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(file)
}

// MiroirTestDir is the local copy of the miroir-deployment MiroirTest
// instance directory (entity a311f363-e238-4203-bdfc-29e8c160c26b),
// mirrored under go/packages/.
func MiroirTestDir() string {
	return filepath.Join(packageDir(), "..", "packages", "miroir-test-app_deployment-miroir", "assets", "miroir_data", "a311f363-e238-4203-bdfc-29e8c160c26b")
}

// MiroirTestFile returns the JSON path for a MiroirTest instance uuid.
func MiroirTestFile(uuid string) string {
	return filepath.Join(MiroirTestDir(), uuid+".json")
}
