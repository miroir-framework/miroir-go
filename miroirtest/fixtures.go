package miroirtest

import (
	"fmt"

	"github.com/miroir-framework/miroir/go/jzod"
)

func resolveEnvironment(ref string) (any, error) {
	switch ref {
	case "defaultMiroirModelEnvironment":
		return jzod.DefaultMiroirModelEnvironment(), nil
	case "defaultMetaModelEnvironment":
		return jzod.DefaultMiroirModelEnvironment(), nil
	case "defaultMiroirMetaModel":
		return jzod.DefaultMiroirMetaModel(), nil
	default:
		return nil, fmt.Errorf("Unknown functionCallTest environmentRef: %s", ref)
	}
}

func resolveFixture(ref string) (any, error) {
	switch ref {
	case "miroirFundamentalJzodSchema":
		return jzod.FundamentalSchema(), nil
	default:
		return nil, fmt.Errorf("Unknown functionCallTest fixtureRef: %s", ref)
	}
}

func resolveFixtureProperty(fixture any, property string) (any, error) {
	key := property
	if key == "" {
		key = "domainState"
	}
	if m, ok := fixture.(map[string]any); ok {
		if v, exists := m[key]; exists {
			return v, nil
		}
		if key == "domainState" {
			return fixture, nil
		}
	}
	if key == "domainState" {
		return fixture, nil
	}
	return nil, fmt.Errorf("Unknown fixture property: %s", key)
}
