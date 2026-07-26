// Package steps implements the godog step definitions for the search-query
// feature. It drives every scenario exclusively through the feature's
// Dispatcher (see acceptance.World.SendAndWait) and asserts on outcomes via
// the fake AuditGateway's Result channel and the fake ItemGateway's call
// count.
package steps

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/cucumber/godog"

	"github.com/esdatalabs/troventory/internal/search/acceptance"
	"github.com/esdatalabs/troventory/internal/search/entities"
	"github.com/esdatalabs/troventory/internal/search/services/query"
)

// RegisterQuerySteps wires every Given/When/Then for query.feature against
// the shared World w.
func RegisterQuerySteps(sc *godog.ScenarioContext, w *acceptance.World) {
	// Background
	sc.Given(`^a location named "([^"]+)" with no parent$`, func(name string) error {
		w.Locations.Seed(name, "")
		return nil
	})

	sc.Given(`^a location named "([^"]+)" with parent "([^"]+)"$`, func(name, parent string) error {
		w.Locations.Seed(name, parent)
		return nil
	})

	sc.Given(`^an item described as "([^"]+)" with category "([^"]+)" exists in the catalog assigned to location "([^"]+)" with a current value of "([^"]+)"$`,
		func(description, category, location, value string) error {
			w.Items.Seed(description, category, location)
			cents, currency := mustParseMoney(value)
			w.Values.Seed(description, cents, currency)
			return nil
		})

	// Given (setup for archival/idempotency scenarios)
	sc.Given(`^the item "([^"]+)" has been archived$`, func(description string) error {
		w.Items.Archive(description)
		return nil
	})

	sc.Given(`^a search request for items whose description contains "([^"]+)" with reference "([^"]+)"$`, func(description, reference string) error {
		w.StagedByRef[reference] = query.Command{
			CorrelationID:       reference,
			Reference:           reference,
			DescriptionContains: description,
		}
		return nil
	})

	// When
	sc.When(`^I search for items whose description contains "([^"]+)"$`, func(description string) error {
		return sendSearch(w, query.Command{DescriptionContains: description})
	})

	sc.When(`^I filter items by category "([^"]+)"$`, func(category string) error {
		return sendSearch(w, query.Command{Category: category})
	})

	sc.When(`^I filter items by location "([^"]+)"$`, func(location string) error {
		return sendSearch(w, query.Command{LocationName: location})
	})

	sc.When(`^I filter items with a current value between "([^"]+)" and "([^"]+)"$`, func(min, max string) error {
		return sendSearch(w, valueRangeCommand(min, max))
	})

	sc.When(`^I search for items whose description contains "([^"]+)" with category "([^"]+)" and a current value between "([^"]+)" and "([^"]+)"$`,
		func(description, category, min, max string) error {
			cmd := valueRangeCommand(min, max)
			cmd.DescriptionContains = description
			cmd.Category = category
			return sendSearch(w, cmd)
		})

	sc.When(`^I filter items by category "([^"]+)" with a blank description filter$`, func(category string) error {
		return sendSearch(w, query.Command{Category: category, DescriptionContains: ""})
	})

	sc.When(`^I search with no description, category, location, or value filter$`, func() error {
		return sendSearch(w, query.Command{})
	})

	sc.When(`^I attempt to filter items with a current value between "([^"]+)" and "([^"]+)"$`, func(min, max string) error {
		return sendSearch(w, valueRangeCommand(min, max))
	})

	sc.When(`^I attempt to filter items by a location that does not exist$`, func() error {
		return sendSearch(w, query.Command{LocationName: "a location that does not exist"})
	})

	sc.When(`^the request with reference "([^"]+)" is submitted$`, func(reference string) error {
		return submitStaged(w, reference)
	})

	sc.When(`^the same request with reference "([^"]+)" is submitted again$`, func(reference string) error {
		return submitStaged(w, reference)
	})

	// Then
	sc.Then(`^the search results (?:also )?include (.+)$`, func(rest string) error {
		if w.LastResult.Err != nil {
			return fmt.Errorf("expected the search to succeed, but it failed: %v", w.LastResult.Err)
		}

		rest = stripBecauseClause(rest)
		exact := strings.HasPrefix(rest, "exactly ")
		rest = strings.TrimPrefix(rest, "exactly ")

		want := extractQuoted(rest)
		if len(want) == 0 {
			return fmt.Errorf("no expected descriptions found in %q", rest)
		}

		if exact {
			if len(w.LastResult.Matches) != len(want) {
				return fmt.Errorf("expected exactly %v, got %v", want, descriptions(w.LastResult.Matches))
			}
		}

		for _, description := range want {
			if !containsDescription(w.LastResult.Matches, description) {
				return fmt.Errorf("expected search results to include %q, got %v", description, descriptions(w.LastResult.Matches))
			}
		}
		return nil
	})

	sc.Then(`^the search results do not include (.+)$`, func(rest string) error {
		if w.LastResult.Err != nil {
			return fmt.Errorf("expected the search to succeed, but it failed: %v", w.LastResult.Err)
		}

		unwanted := extractQuoted(stripBecauseClause(rest))
		for _, description := range unwanted {
			if containsDescription(w.LastResult.Matches, description) {
				return fmt.Errorf("expected search results not to include %q, got %v", description, descriptions(w.LastResult.Matches))
			}
		}
		return nil
	})

	sc.Then(`^the search returns no results$`, func() error {
		if w.LastResult.Err != nil {
			return fmt.Errorf("expected the search to succeed with no matches, but it failed: %v", w.LastResult.Err)
		}
		if len(w.LastResult.Matches) != 0 {
			return fmt.Errorf("expected no search results, got %v", descriptions(w.LastResult.Matches))
		}
		return nil
	})

	sc.Then(`^no results are returned$`, func() error {
		if len(w.LastResult.Matches) != 0 {
			return fmt.Errorf("expected no results, got %v", descriptions(w.LastResult.Matches))
		}
		return nil
	})

	sc.Then(`^the search is rejected because the minimum value exceeds the maximum value$`, func() error {
		return assertLastErrIs(w, entities.ErrInvalidValueRange)
	})

	sc.Then(`^the search is rejected because the given location cannot be found$`, func() error {
		return assertLastErrIs(w, entities.ErrLocationNotFound)
	})

	sc.Then(`^the search results are ordered (.+)$`, func(rest string) error {
		if w.LastResult.Err != nil {
			return fmt.Errorf("expected the search to succeed, but it failed: %v", w.LastResult.Err)
		}

		want := extractQuoted(rest)
		got := descriptions(w.LastResult.Matches)
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("expected search results ordered %v, got %v", want, got)
		}
		return nil
	})

	sc.Then(`^both submissions report the same single matching item "([^"]+)"$`, func(description string) error {
		results, ok := w.ResultsByRef[w.LastReference]
		if !ok || len(results) < 2 {
			return fmt.Errorf("expected two recorded results for reference %q, got %d", w.LastReference, len(results))
		}

		first, second := results[0], results[1]
		if first.Err != nil || second.Err != nil {
			return fmt.Errorf("expected both submissions to succeed, got errors %v / %v", first.Err, second.Err)
		}
		if len(first.Matches) != 1 || first.Matches[0].Description != description {
			return fmt.Errorf("expected the first submission to match exactly %q, got %v", description, descriptions(first.Matches))
		}
		if !reflect.DeepEqual(first, second) {
			return fmt.Errorf("expected both submissions to report identical results: first %+v, second %+v", first, second)
		}
		return nil
	})

	sc.Then(`^the search is only carried out once$`, func() error {
		if calls := w.Items.FindAllCallCount(); calls != 1 {
			return fmt.Errorf("expected the search to be carried out exactly once, was carried out %d times", calls)
		}
		return nil
	})
}

// sendSearch assigns a fresh correlation ID to cmd and sends it through the
// Dispatcher.
func sendSearch(w *acceptance.World, cmd query.Command) error {
	cmd.CorrelationID = w.NextRef()
	_, err := w.SendAndWait(cmd)
	return err
}

// submitStaged sends the Command previously staged under reference through
// the Dispatcher, recording reference as the World's LastReference for the
// "both submissions report the same ..." assertion.
func submitStaged(w *acceptance.World, reference string) error {
	cmd, ok := w.StagedByRef[reference]
	if !ok {
		return fmt.Errorf("no staged request for reference %q", reference)
	}
	w.LastReference = reference
	_, err := w.SendAndWait(cmd)
	return err
}

// valueRangeCommand builds a Command carrying a current-value range filter
// parsed from decimal dollar strings, assuming USD for both bounds.
func valueRangeCommand(min, max string) query.Command {
	minCents, currency := mustParseMoney(min)
	maxCents, _ := mustParseMoney(max)
	return query.Command{
		MinValueCents: &minCents,
		MaxValueCents: &maxCents,
		Currency:      currency,
	}
}

// assertLastErrIs confirms the most recent Result recorded via the fake
// AuditGateway failed with the given sentinel error.
func assertLastErrIs(w *acceptance.World, target error) error {
	if w.LastResult.Err == nil {
		return fmt.Errorf("expected the last result to fail with %v, but it succeeded", target)
	}
	if !errors.Is(w.LastResult.Err, target) {
		return fmt.Errorf("expected the last result's error to be %v, got %v", target, w.LastResult.Err)
	}
	return nil
}

// becauseClause matches a trailing ", because ... is nested under ..."
// justification clause so it can be stripped before extracting the quoted
// item descriptions a Then step actually expects to see in the results.
var becauseClause = regexp.MustCompile(`(?i),\s*because\b.*$`)

// stripBecauseClause removes a trailing ", because ..." justification
// clause from s.
func stripBecauseClause(s string) string {
	return becauseClause.ReplaceAllString(s, "")
}

// quotedString matches a single double-quoted substring.
var quotedString = regexp.MustCompile(`"([^"]+)"`)

// extractQuoted returns every double-quoted substring in s, in order.
func extractQuoted(s string) []string {
	matches := quotedString.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// descriptions returns the Description of every item in matches, in order,
// for building readable failure messages.
func descriptions(matches []entities.Item) []string {
	out := make([]string, 0, len(matches))
	for _, item := range matches {
		out = append(out, item.Description)
	}
	return out
}

// containsDescription reports whether matches contains an item described as
// description.
func containsDescription(matches []entities.Item, description string) bool {
	for _, item := range matches {
		if item.Description == description {
			return true
		}
	}
	return false
}

// mustParseMoney converts a decimal dollar string like "450.00" into an
// integer minor-unit (cents) amount and a fixed "USD" currency code, per
// ARCHITECTURE.md §6 (never float64 for money). It panics on malformed
// input, which only ever comes from literal strings in this test's own step
// definitions.
func mustParseMoney(dollars string) (cents int64, currency string) {
	c, err := parseCents(dollars)
	if err != nil {
		panic(fmt.Sprintf("mustParseMoney(%q): %v", dollars, err))
	}
	return c, "USD"
}

// parseCents converts a decimal dollar string like "450.00" into an integer
// number of cents, without ever going through float64.
func parseCents(dollars string) (int64, error) {
	parts := strings.SplitN(strings.TrimSpace(dollars), ".", 2)

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse whole dollars %q: %w", dollars, err)
	}

	var cents int64
	if len(parts) == 2 {
		frac := parts[1]
		switch {
		case len(frac) == 1:
			frac += "0"
		case len(frac) > 2:
			frac = frac[:2]
		}
		c, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse fractional cents %q: %w", dollars, err)
		}
		cents = c
	}

	return whole*100 + cents, nil
}
