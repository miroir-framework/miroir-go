# 255 — Go Jzod definitions and runtime typecheck

> Analysis: port the Jzod *language* (Miroir bootstrap schema) and Miroir’s native `jzodTypeCheck`
> to Go so JSON values can be declared and checked at runtime. First proving case: bootstrap
> self-parse. Does not port the MiroirTest machine (#256) or transformer apply (#257).

Related issue: https://github.com/miroir-framework/miroir/issues/255
Parent: [#254](https://github.com/miroir-framework/miroir/issues/254) · Unblocks: #256, #257
Related analyses: [`../254-FEATURE-go-backend/analysis.md`](../254-FEATURE-go-backend/analysis.md)

Key sources:
- [`packages/miroir-test-app_deployment-miroir/assets/miroir_data/5e81e1b9-38be-487c-b3e5-53796c57fccf/1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json`](../../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/5e81e1b9-38be-487c-b3e5-53796c57fccf/1e8dab4b-65a3-4686-922e-ce89a2d62aa9.json)
- [`packages/miroir-core/src/1_core/jzod/jzodTypeCheck.ts`](../../../packages/miroir-core/src/1_core/jzod/jzodTypeCheck.ts)
- [`packages/miroir-core/src/0_interfaces/1_core/jzodTypeCheckInterface.ts`](../../../packages/miroir-core/src/0_interfaces/1_core/jzodTypeCheckInterface.ts)
- [`packages/miroir-test-app_deployment-miroir/assets/miroir_data/a311f363-e238-4203-bdfc-29e8c160c26b/3aff508a-8a9f-4384-ba50-cc696411eba5.json`](../../../packages/miroir-test-app_deployment-miroir/assets/miroir_data/a311f363-e238-4203-bdfc-29e8c160c26b/3aff508a-8a9f-4384-ba50-cc696411eba5.json)
- Sibling `jzod/src/JzodInterface.ts` (`jzodBootstrapElementSchema`), `jzod/src/facade.ts` (`jzodToZod`)

**Document role:** analysis and architectural decision record.
**Status:** implemented and reviewed (2026-09-03). `go/jzod` `TypeCheck` + inventory, all 42 `3aff508a-…` leaves, bootstrap self-parse.

---

## Decision record

| Decision | Choice |
|---|---|
| D1 — Package | **`go/jzod`** inside module `go/` (#254 D2) |
| D2 — Typecheck semantics | **Port full `ResolvedJzodSchemaReturnType`** (`status`, `rawSchema`, `resolvedSchema`, `valuePath`, `typePath`, `schemaReferenceName`, `keyMap`; errors also `error`, `rawJzodSchemaType`, `value`). Not Zod `parse`. |
| D3 — Schema source | **Load `jzodMiroirBootstrapSchema` JSON** (uuid `1e8dab4b-…`). Do not transpile sibling `jzodBootstrapElementSchema`. |
| D4 — Value / schema representation | **`encoding/json` → `any` / typed accessors**. No generated Go structs as the product. |
| D5 — Proof in this issue | **`go test` against real JSON**: (1) first leaves of `jzodTypeCheck_TransformerTestSuite` calling `TypeCheck` directly; (2) bootstrap self-parse. Not a MiroirTest runner. |
| D6 — Constructor scope | **Implement what proofs need.** Types TS **stub-accepts** (`ok`, schema echoed: `undefined`, `never`, `unknown`, `void`, `intersection`, `promise`, `set`, `function`, `map`, `lazy` at `jzodTypeCheck.ts:2241-2270`) must stub-accept in Go — **not** fail closed. Fail closed only for a `type` string TS does not handle. |
| D7 — `modelEnvironment` | **Register bootstrap `MlSchema` `1e8dab4b-…` in the lookup chain** (see D7 detail). `defaultMiroirModelEnvironment` cannot resolve that `absolutePath` (`defaultMiroirMetaModel.jzodSchemas` is `[]` at `Model.ts:302`). Suite `3aff508a-…` uses **no** `absolutePath`. |
| D8 — `resolveConditionalSchema` | **Skip unless `currentDefaultValue` and `reduxDeploymentsState` are present** (`jzodTypeCheck.ts:941-955`). Empty `currentValuePath: []` is truthy in JS and does **not** skip. Unit leaves omit the two optionals. |
| D9 — `valueToJzod` | **Port the sliver** for `type: "any"`: `buildAnyObjectEntry` **and** `buildAnySubnodeKeyMap` (`jzodTypeCheck.ts:103`, `:141`). Not the full sibling product. |

**Rationale:** #257’s `jzodTypeCheck` transformer tests compare full `expectedValue` objects. A boolean `valid/invalid` API would force a rewrite. Loading the Miroir JSON avoids drifting from the asset that “parses itself.”

### D2 — Typecheck semantics

**Status:** Accepted — `jzodTypeCheck` shape.

| Option | Mechanism | Pros | Cons |
|---|---|---|---|
| **D2-a. Port `jzodTypeCheck`** ★ | Go `TypeCheck(schema, value, …) Result` | Same expected values as suite `3aff508a-…` | Large (TS file ~2340 lines) |
| D2-b. `jzodToZod` + Zod | Convert then parse | Matches sibling README | Different errors; no `keyMap`; CGO or rewrite of Zod |
| D2-c. Accept/reject only | `error` or `nil` | Small | Cannot pass existing unit expected values |

**Decision:** D2-a. Compare the fields each leaf’s `expectedValue` actually contains. Exact `keyMap` parity for every `any`/union edge is a **deepening** target after `status` + `resolvedSchema` + `rawSchema` + paths.

### D7 — `modelEnvironment` for bootstrap

**Status:** Accepted — register `1e8dab4b-…` as an `MlSchema` in the absolute-reference list.

TS lookup (`jzodResolveSchemaReferenceInContext.ts:104-119`):

```
absoluteReferences = [
  miroirFundamentalJzodSchema,
  ...currentModel.jzodSchemas,
  ...miroirMetaModel.jzodSchemas,
]
target = absoluteReferences.find(s => s.uuid == absolutePath)?.definition.context[relativePath]
```

`defaultMiroirModelEnvironment` never puts `1e8dab4b-…` in that list. Deployment bootstrap registers it separately (`miroir-test-app_deployment-miroir/src/Model.ts:294`, `jzodSchemas: [jzodSchemajzodMiroirBootstrapSchema]`).

Go `ModelEnvironment` for Slice 7: one registered `{ uuid: 1e8dab4b-…, definition.context }` plus whatever local `context` the schemaReference carries. Do **not** load `fe9b7d99-…` unless a reused leaf’s `absolutePath` is that uuid (none in `3aff508a-…`).

Six `fe9b7d99-…` strings inside bootstrap `jzodBaseObject.tag` editor metadata (`1e8dab4b-…json`) are annotations, not suite payloads; self-parse need not resolve them (`tag` values are simple; nested transformer fields are `type: "any"`).

### D3 — Which bootstrap

**Status:** Accepted — Miroir JSON `1e8dab4b-…`.

Enumerated 2026-09-03 from that file:

- `name`: `jzodMiroirBootstrapSchema`
- `parentUuid`: `5e81e1b9-38be-487c-b3e5-53796c57fccf` (Entity `MlSchema`)
- Root: `{ type: "schemaReference", definition: { absolutePath: "1e8dab4b-…", relativePath: "jzodElement" } }`
- **25** context keys: `jzodArray`, `jzodAttributeDateValidations`, `jzodAttributeNumberValidations`, `jzodAttributePlainDateWithValidations`, `jzodAttributePlainNumberWithValidations`, `jzodAttributePlainStringWithValidations`, `jzodAttributeStringValidations`, `jzodBaseObject`, `jzodElement`, `jzodEnum`, `jzodEnumAttributeTypes`, `jzodEnumElementTypes`, `jzodFunction`, `jzodIntersection`, `jzodLazy`, `jzodLiteral`, `jzodMap`, `jzodObject`, `jzodPlainAttribute`, `jzodPromise`, `jzodRecord`, `jzodReference`, `jzodSet`, `jzodTuple`, `jzodUnion`
- `jzodElement`: union, discriminator field `type`, **18** `schemaReference` branches (list in §3.2)
- `jzodEnumAttributeTypes`: `any`, `bigint`, `boolean`, `date`, `never`, `null`, `number`, `string`, `uuid`, `undefined`, `unknown`, `void`
- `jzodEnumElementTypes`: `array`, `date`, `enum`, `function`, `lazy`, `literal`, `intersection`, `map`, `number`, `object`, `promise`, `record`, `schemaReference`, `set`, `string`, `tuple`, `union`

Sibling `jzod/src/JzodInterface.ts` `jzodBootstrapElementSchema` is a **different artefact** (TS constant, richer `tag`/`metaSchema` comments). Copying it would fork the language Miroir actually stores.

### D5 — Proof without #256

**Status:** Accepted — `go test` imports the real suite JSON and calls `TypeCheck` with each leaf’s `transformer.mlSchema` / `transformer.valueObject`, comparing to `expectedValue` (or `status`+`resolvedSchema` first).

This is the skill’s “import real applicative assets, not fixture copies.” The suite is a `transformerTest` list; #255 does **not** interpret `transformerType`. #257 will run the same file through the machine.

---

## 1. Goals

1. **Declare Jzod in Go** — In order to manipulate ML JSON as a \<Go backend author\>, I can unmarshal a `JzodElement` (including the bootstrap schema) without a TypeScript build.
2. **Typecheck values** — In order to reject bad instances as a \<runtime maintainer\>, I can call `TypeCheck` and get an ok/error result aligned with Miroir `jzodTypeCheck`.
3. **Bootstrap self-parse** — In order to trust the language is closed as a \<schema author\>, I can typecheck `jzodMiroirBootstrapSchema.definition` against itself (with `1e8dab4b-…` registered in the env) and get `status: "ok"`. This is a **new** Go proof — TS has no `jzodTypeCheck` self-parse test (only the sibling `jzodToZod` README demo).

## 2. Non-goals

- MiroirTest runner / `functionCallTest` dispatch (#256).
- `transformer_extended_apply_wrapper` and the 45 TransformerDefinitions (#257).
- `jzodToZod`, `zodToJzod`, `jzod-ts` codegen, Jzod→JSON-Schema product (`jzodToJsonSchema` suite).
- Full `defaultMiroirModelEnvironment` / carry-on / fundamental schema generation (`getMiroirFundamentalJzodSchema.ts`).
- `resolveConditionalSchema` engine unless a #255 leaf requires the `reduxDeploymentsState` path.
- Real value-checking for TS stub-accept types (`function`, `promise`, `map`, `set`, `lazy`, `intersection`, …) beyond echoing the schema with `status: "ok"` (`jzodTypeCheck.ts:2241-2270`). Do **not** fail closed on those types.

## 3. Current state

### 3.1 Two typecheck paths (aligned vs wrong target)

**Wrong target for this issue:** sibling `jzodToZod` (`jzod/src/facade.ts`) returns a Zod schema; README bootstrap is `.parse(jzodBootstrapElementSchema)`. Tests: `jzod/tests/jzodToZod.test.ts` (conversion vs Zod JSON-Schema, not Miroir `ResolvedJzodSchemaReturnType`).

**Right target:** `export function jzodTypeCheck` at `jzodTypeCheck.ts:871`:

```text
jzodTypeCheck(
  mlSchema, valueObject,
  currentValuePath, currentTypePath,
  modelEnvironment, relativeReferenceJzodContext,
  currentDefaultValue?, reduxDeploymentsState?,
  deploymentUuid?, rootObject?, schemaReferenceName?
) → ResolvedJzodSchemaReturnType
```

Comment at `:854-860`: removes unions and references, node-for-node, checks the value.

`jzodTypeCheckTransformer` (`:2297-2335`) unpacks transformer fields and **always** passes `defaultMiroirModelEnvironment` (comment: `TODO: use proper model environment`). Unit leaves in `3aff508a-…` set `relativeReferenceJzodContext: {}` and empty paths; they do not pass `reduxDeploymentsState`.

### 3.2 Bootstrap constructors (aligned — enumerated)

`jzodElement` union branches (relativePath), in file order:

1. `jzodArray` 2. `jzodPlainAttribute` 3. `jzodAttributePlainDateWithValidations` 4. `jzodAttributePlainNumberWithValidations` 5. `jzodAttributePlainStringWithValidations` 6. `jzodEnum` 7. `jzodFunction` 8. `jzodLazy` 9. `jzodLiteral` 10. `jzodIntersection` 11. `jzodMap` 12. `jzodObject` 13. `jzodPromise` 14. `jzodRecord` 15. `jzodReference` 16. `jzodSet` 17. `jzodTuple` 18. `jzodUnion`

Object `extend` is a `schemaReference` (used by almost every constructor via `jzodBaseObject`). Self-parse therefore requires **reference resolution + object extend + discriminated union**, not only primitives.

### 3.3 Existing typecheck corpus (aligned — enumerated)

Suite `jzodTypeCheck_TransformerTestSuite` uuid `3aff508a-8a9f-4384-ba50-cc696411eba5`, export `miroirTest_jzodTypeCheck`. Vitest loader: `packages/miroir-core/tests/1_core/jzod/jzodTypeCheck.test.ts` → `runMiroirCoreTestSuite`. The loader **skips** when the test-runner **file-pattern argument** contains `resolveConditionalSchema` (`jzodTypeCheck.test.ts:8-15`) — not Vitest’s internal file filter.

**42** `transformerTest` leaves, labels in file order:

`test010_literal`, `test020_string`, `test022_boolean_true`, `test024_boolean_false`, `test030_schemaReference`, `test040_simple_object`, `test050_object_with_union`, `test060_recursive_object`, `test070`, `test120_simple_union`, `test130_any_string`, `test140_any_number`, `test150_any_object`, `test160_any_object_of_object`, `test170_any_array`, `test180_any_null`, `test181_any_boolean`, `test182_any_empty_object`, then a second series with **prefix-colliding** labels (`test010` ≠ `test010_literal`, skips `test023`, then `test024`…`test030`, `test100`, `test110`, `test120`, `test130`). Identity is **document order + parent path**, not label. TS filter uses exact label match (`MiroirTransformerTestTools.ts:149`).

First leaf `test010_literal`: `mlSchema = { type: "literal", definition: "myLiteral" }`, `valueObject = "myLiteral"`, `expectedValue.status = "ok"`, `resolvedSchema` equals raw literal, `keyMap[""]` mirrors that.

`test030_schemaReference`: local `context.a = { type: "string" }`, `definition.relativePath = "a"`, value `"myString"`. **No** `absolutePath`. Type path becomes `["ref:a"]`.

### 3.4 Dependencies of `jzodTypeCheck` (aligned)

| Callee | File | Needed for bootstrap / first leaves? |
|---|---|---|
| `resolveJzodSchemaReferenceInContext` | `jzodResolveSchemaReferenceInContext.ts:57` | **Yes** (`schemaReference`, `extend`, bootstrap `absolutePath`) |
| `recursiveResolveJzodSchemaReferenceInContext` | same file `:150`; called from `jzodTypeCheck.ts:325`, `:984` | **Yes** |
| `jzodUnion_recursivelyUnfold` | `jzodUnion_RecursivelyUnfold.ts` | **Yes** (unions; bootstrap `jzodElement`) |
| `jzodObjectFlatten` | `jzodObjectFlatten.ts`; called `:248`, `:1057` | **Yes** (object check + union flatten) |
| `resolveConditionalSchema` | `resolveConditionalSchema.ts` | **No** for unit leaves / bootstrap: they omit `currentDefaultValue` and `reduxDeploymentsState` |
| `getObjectUnionDiscriminatorValuesFromResolvedSchema` | `getObjectUnionDiscriminatorValues.ts` | Discriminated object unions |
| `valueToJzod` | `@miroir-framework/jzod` | `type: "any"` keymap (`buildAnyObjectEntry` `:103`, `buildAnySubnodeKeyMap` `:141`) |
| `defaultMiroirModelEnvironment` | `Model.ts:126` | **Cannot** resolve `1e8dab4b-…`. Suite leaves need no absolutePath. Slice 7 must register the bootstrap `MlSchema` (D7). |

`jzodTypeCheck` switch (`:981+`) **fully** handles: `schemaReference`, `object`, `union`, `record`, `literal`, `enum`, `tuple`, `array`, `any`, `uuid`, `string`, `number`, `bigint`, `boolean`, `date`. **Stub-accept** (`status: "ok"`, schema echoed, `:2241-2270`): `undefined`, `never`, `unknown`, `void`, `intersection`, `promise`, `set`, `function`, `map`, `lazy`. Bootstrap self-parse typechecks those constructors **as values of the meta-schema**. A top-level `{ type: "function", … }` value-check in TS is stub-ok; Go must match.

Another TS consumer (not a #255 proof): `checkModelValidationInstance` in `ModelValidationTools.ts:158` calls `jzodTypeCheck` with empty paths.

Exported helpers in the same file (`unionObjectChoices`, `selectUnionBranchFromDiscriminator`, …) already have **`functionCallTest`** suites (`unionObjectChoices` `14319c8e-…`, `selectUnionBranchFromDiscriminator` `84e67b10-…`). Those suites are **#256** (machine) + optional extra #255 `go test` if we port the helpers as public Go functions. Not required to close #255 if `TypeCheck` internals stay unexported.

### 3.5 `fe9b7d99-…` vs `1e8dab4b-…` (do not conflate)

| Uuid | What it is |
|---|---|
| `1e8dab4b-65a3-4686-922e-ce89a2d62aa9` | Stored `MlSchema` instance — Jzod language bootstrap |
| `fe9b7d99-f216-44de-bb6e-60e1a1ebb739` | `miroirFundamentalJzodSchemaUuid` in `getMiroirFundamentalJzodSchemaHelpers.ts:26`; composed by `getMiroirFundamentalJzodSchema.ts` |

Typecheck **interface** Jzod literals in `jzodTypeCheckInterface.ts` (`keyMapEntry`, `resolvedJzodSchemaReturnTypeOK`) reference `fe9b7d99-…`. That does not mean bootstrap self-parse must load the fundamental schema. Self-parse uses `absolutePath: 1e8dab4b-…`.

## 4. Key reuse

| Piece | Location |
|-------|----------|
| Bootstrap JSON | `1e8dab4b-65a3-4686-922e-ce89a2d62aa9` |
| Typecheck suite JSON | `3aff508a-8a9f-4384-ba50-cc696411eba5` |
| Result types (TS) | `jzodTypeCheckInterface.ts` `ResolvedJzodSchemaReturnType*` |
| Transformer wrapper (later #257) | `jzodTypeCheckTransformer` `:2297`; definition `a3f7b5c2-1e8d-4a9b-9c7e-6f2d3e8a1b5c` |
| `ANY_SCHEMA` / implicit union | `jzodTypeCheck.ts:56-87` |

## 5. Proposals / options

| # | Proposal | Impact | Effort | Verdict |
|---|---|---|---|---|
| 1 | Interpreter in `go/jzod` + `go test` on suite JSON + bootstrap | Unblocks #256/#257 | High | **adopt** |
| 2 | Generate Go types from bootstrap and typecheck via `encoding/json` tags only | Cannot express unions/refs | Med | reject |
| 3 | Call TS `jzodTypeCheck` via Node child process | Not a Go port | Low | reject |

---

## Next step

Implemented and reviewed. Child issues #256 / #257 are realized on the same JSON assets.
