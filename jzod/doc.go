// Package jzod is the Go port of Miroir jzodTypeCheck and related Jzod helpers.
//
// It interprets bootstrap / fundamental schema JSON from the mirrored copies
// under go/packages/ (jzodMiroirBootstrapSchema, miroirFundamentalJzodSchema)
// as generated.JzodElement values and returns ResolvedJzodSchemaReturnType-shaped
// [Result] values. There is no Zod binding.
package jzod
