# miroir-go

Go implementation of the Miroir Meta-Language runtime: Jzod typecheck, the unit MiroirTest machine, and transformer apply. It is a library plus a small codegen CLI. There is no HTTP server in this repo yet.

Module path: `github.com/miroir-framework/miroir/go`. Requires **Go 1.22+**. Run every command below from this repository root.

## Tests

```bash
go test ./...
```

Package slices:

```bash
go test ./jzod          # TypeCheck, bootstrap self-parse, suite 3aff508a-…
go test ./jzodgen       # Jzod → Go type emitter and bootstrap file freshness
go test ./miroirtest    # functionCallTest + transformerTest JSON
```

Useful filters:

```bash
go test ./jzod -run TestBootstrapSelfParse
go test ./jzod -run TestTypeCheck
go test ./jzodgen -run TestJzodToGoType
go test ./jzodgen -run TestGenerateBootstrapPackage
```

Tests load the mirrored Miroir JSON under `packages/` and Go-local suites under `tests/`. They do not start a process.

## Regenerate Go types

`jzod/generated/jzod_miroir_bootstrap.go` is emitted from `jzodMiroirBootstrapSchema` (`packages/miroir-test-app_deployment-miroir/assets/miroir_data/5e81e1b9-38be-487c-b3e5-53796c57fccf/1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json`). Do not edit the generated file by hand.

After changing `jzodgen` or that bootstrap JSON:

```bash
go generate ./jzodgen
```

Equivalent:

```bash
go run ./cmd/genjzod
```

Then `go test ./jzodgen -run TestGeneratedBootstrapFileMatches` checks that the committed file matches generation.

## Build and launch the binary

The CLI is `cmd/genjzod`. It writes the bootstrap types file and prints the path.

Run without installing:

```bash
go run ./cmd/genjzod
```

Build then run (from the module root):

```bash
go build -o genjzod ./cmd/genjzod
./genjzod
```

On Windows (cmd or PowerShell), the binary is `genjzod.exe`. Git Bash can use `./genjzod`.

`go build ./...` compiles every package; it does not produce a server binary.
