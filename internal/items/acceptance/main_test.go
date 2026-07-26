package acceptance_test

import (
	"context"
	"testing"

	"github.com/cucumber/godog"

	"github.com/esdatalabs/troventory/internal/items/acceptance"
	"github.com/esdatalabs/troventory/internal/items/acceptance/steps"
)

// TestFeatures runs every scenario in acceptance/features against a fresh
// World per scenario. Run with: go test ./internal/items/acceptance/...
func TestFeatures(t *testing.T) {
	world := acceptance.NewWorld()

	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				world.Reset()
				return ctx, nil
			})
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				world.Close()
				return ctx, nil
			})

			steps.RegisterCatalogSteps(sc, world)
			steps.RegisterEnrichSteps(sc, world)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
