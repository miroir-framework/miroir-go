// Package miroirtest is the Go unit MiroirTest machine.
//
// It loads MiroirTest JSON from the mirrored copy under go/packages/
// (entity a311f363-…). [RunSuite] dispatches functionCallTest and transformerTest
// leaves; queryTest, actionTest, and runnerTest fail closed.
package miroirtest
