// Package steps implements the godog step definitions for the manage
// locations feature. It drives every scenario exclusively through the
// feature's Dispatcher (see acceptance.World.SendAndWait) and asserts on
// outcomes via the fake AuditGateway's Result channel and the fake
// StorageGateway's state.
package steps

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"

	"github.com/esdatalabs/troventory/internal/locations/acceptance"
	"github.com/esdatalabs/troventory/internal/locations/entities"
	"github.com/esdatalabs/troventory/internal/locations/services/manage"
)

// RegisterManageSteps wires every Given/When/Then for manage.feature against
// the shared World w.
func RegisterManageSteps(sc *godog.ScenarioContext, w *acceptance.World) {
	// Given
	sc.Given(`^a location named "([^"]+)" with no parent$`, func(name string) error {
		return seedLocation(w, name, nil)
	})

	sc.Given(`^a location named "([^"]+)" with parent "([^"]+)"$`, func(name, parent string) error {
		return seedLocation(w, name, &parent)
	})

	sc.Given(`^the location "([^"]+)" has been archived$`, func(name string) error {
		res, err := w.SendAndWait(manage.Command{
			CorrelationID: w.NextRef(),
			Action:        manage.ActionArchive,
			TargetName:    name,
		})
		if err != nil {
			return err
		}
		if res.Err != nil {
			return fmt.Errorf("archive %q as scenario setup: %w", name, res.Err)
		}
		return nil
	})

	sc.Given(`^a create-location request for "([^"]+)" under "([^"]+)" with reference "([^"]+)"$`, func(name, parent, reference string) error {
		w.StagedByRef[reference] = manage.Command{
			CorrelationID: reference,
			Reference:     reference,
			Action:        manage.ActionCreate,
			Name:          name,
			ParentName:    &parent,
		}
		return nil
	})

	// When
	sc.When(`^I create a location named "([^"]+)" with no parent$`, func(name string) error {
		return whenCreate(w, name, nil)
	})

	sc.When(`^I create a location named "([^"]+)" with parent "([^"]+)"$`, func(name, parent string) error {
		return whenCreate(w, name, &parent)
	})

	sc.When(`^I create a location named "([^"]+)" with a parent that does not exist$`, func(name string) error {
		missing := "a parent that does not exist"
		return whenCreate(w, name, &missing)
	})

	sc.When(`^I rename "([^"]+)" to "([^"]+)"$`, func(target, newName string) error {
		w.Snapshot(target)
		_, err := w.SendAndWait(manage.Command{
			CorrelationID: w.NextRef(),
			Action:        manage.ActionRename,
			TargetName:    target,
			Name:          newName,
		})
		return err
	})

	sc.When(`^I move "([^"]+)" to parent "([^"]+)"$`, func(target, parent string) error {
		w.Snapshot(target)
		_, err := w.SendAndWait(manage.Command{
			CorrelationID: w.NextRef(),
			Action:        manage.ActionMove,
			TargetName:    target,
			ParentName:    &parent,
		})
		return err
	})

	sc.When(`^I move "([^"]+)" to no parent$`, func(target string) error {
		w.Snapshot(target)
		_, err := w.SendAndWait(manage.Command{
			CorrelationID: w.NextRef(),
			Action:        manage.ActionMove,
			TargetName:    target,
			ParentName:    nil,
		})
		return err
	})

	sc.When(`^I (rename|move|archive) a location that does not exist$`, func(action string) error {
		const missing = "a location that does not exist"
		w.Snapshot(missing)
		_, err := w.SendAndWait(genericActionCommand(w, action, missing))
		return err
	})

	sc.When(`^I (rename|move|archive) "([^"]+)"$`, func(action, target string) error {
		w.Snapshot(target)
		_, err := w.SendAndWait(genericActionCommand(w, action, target))
		return err
	})

	sc.When(`^the request with reference "([^"]+)" is submitted$`, func(reference string) error {
		cmd, ok := w.StagedByRef[reference]
		if !ok {
			return fmt.Errorf("no staged request for reference %q", reference)
		}
		_, err := w.SendAndWait(cmd)
		return err
	})

	sc.When(`^the same request with reference "([^"]+)" is submitted again$`, func(reference string) error {
		cmd, ok := w.StagedByRef[reference]
		if !ok {
			return fmt.Errorf("no staged request for reference %q", reference)
		}
		_, err := w.SendAndWait(cmd)
		return err
	})

	// Then
	sc.Then(`^the location "([^"]+)" exists with no parent$`, func(name string) error {
		loc, err := w.Storage.FindByName(context.Background(), name)
		if err != nil {
			return fmt.Errorf("find %q: %w", name, err)
		}
		if loc.Parent != "" {
			return fmt.Errorf("expected %q to have no parent, got %q", name, loc.Parent)
		}
		return nil
	})

	sc.Then(`^the location "([^"]+)" exists with parent "([^"]+)"$`, func(name, parent string) error {
		loc, err := w.Storage.FindByName(context.Background(), name)
		if err != nil {
			return fmt.Errorf("find %q: %w", name, err)
		}
		if loc.Parent != parent {
			return fmt.Errorf("expected %q to have parent %q, got %q", name, parent, loc.Parent)
		}
		return nil
	})

	sc.Then(`^"([^"]+)" is nested three levels deep under "([^"]+)"$`, func(name, ancestor string) error {
		depth, root, err := ancestryDepth(w, name)
		if err != nil {
			return err
		}
		if depth != 3 {
			return fmt.Errorf("expected %q to be nested 3 levels deep, got %d", name, depth)
		}
		if root != ancestor {
			return fmt.Errorf("expected %q's root ancestor to be %q, got %q", name, ancestor, root)
		}
		return nil
	})

	sc.Then(`^the location is not created$`, func() error {
		return assertNotCreated(w, w.AttemptedName)
	})

	sc.Then(`^the location "([^"]+)" still exists with its original name$`, func(name string) error {
		return assertUnchanged(w, name)
	})

	sc.Then(`^the location "([^"]+)" still exists with its original parent$`, func(name string) error {
		return assertUnchanged(w, name)
	})

	sc.Then(`^the location "([^"]+)" is archived$`, func(name string) error {
		loc, err := w.Storage.FindByName(context.Background(), name)
		if err != nil {
			return fmt.Errorf("find %q: %w", name, err)
		}
		if !loc.Archived {
			return fmt.Errorf("expected %q to be archived", name)
		}
		return nil
	})

	sc.Then(`^"([^"]+)" no longer appears among active locations$`, func(name string) error {
		loc, err := w.Storage.FindByName(context.Background(), name)
		if err != nil {
			return fmt.Errorf("find %q: %w", name, err)
		}
		if !loc.Archived {
			return fmt.Errorf("expected %q to no longer be active", name)
		}
		return nil
	})

	sc.Then(`^the location "([^"]+)" is not archived$`, func(name string) error {
		loc, err := w.Storage.FindByName(context.Background(), name)
		if err != nil {
			return fmt.Errorf("find %q: %w", name, err)
		}
		if loc.Archived {
			return fmt.Errorf("expected %q not to be archived", name)
		}
		return nil
	})

	sc.Then(`^the request fails because the parent location cannot be found$`, func() error {
		return assertLastErrIs(w, entities.ErrLocationNotFound)
	})

	sc.Then(`^the request fails because the parent location is archived$`, func() error {
		return assertLastErrIs(w, entities.ErrLocationArchived)
	})

	sc.Then(`^the request fails because a location with that name already exists under that parent$`, func() error {
		return assertLastErrIs(w, entities.ErrDuplicateLocationName)
	})

	sc.Then(`^the request fails because the move would make the location its own descendant$`, func() error {
		return assertLastErrIs(w, entities.ErrCyclicMove)
	})

	sc.Then(`^the request fails because the location still has active children$`, func() error {
		return assertLastErrIs(w, entities.ErrLocationHasActiveChildren)
	})

	sc.Then(`^the request fails because the location cannot be found$`, func() error {
		return assertLastErrIs(w, entities.ErrLocationNotFound)
	})

	sc.Then(`^the request fails because the location is archived$`, func() error {
		return assertLastErrIs(w, entities.ErrLocationArchived)
	})

	sc.Then(`^exactly one location named "([^"]+)" exists under "([^"]+)"$`, func(name, parent string) error {
		children, err := w.Storage.ChildrenOf(context.Background(), parent)
		if err != nil {
			return fmt.Errorf("children of %q: %w", parent, err)
		}
		count := 0
		for _, child := range children {
			if child.Name == name {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("expected exactly one location named %q under %q, found %d", name, parent, count)
		}
		return nil
	})
}

// seedLocation creates a location directly through the Dispatcher (the only
// legitimate entry point) as part of scenario setup, and fails fast if setup
// itself doesn't succeed.
func seedLocation(w *acceptance.World, name string, parent *string) error {
	res, err := w.SendAndWait(manage.Command{
		CorrelationID: w.NextRef(),
		Reference:     w.NextRef(),
		Action:        manage.ActionCreate,
		Name:          name,
		ParentName:    parent,
	})
	if err != nil {
		return err
	}
	if res.Err != nil {
		return fmt.Errorf("seed location %q: %w", name, res.Err)
	}
	return nil
}

// whenCreate stages a create attempt driven through the Dispatcher, snapshots
// prior state so "the location is not created" can tell a pre-existing
// collision apart from a genuinely new name, and records the attempted name
// for that same assertion.
func whenCreate(w *acceptance.World, name string, parent *string) error {
	w.AttemptedName = name
	w.Snapshot(name)
	_, err := w.SendAndWait(manage.Command{
		CorrelationID: w.NextRef(),
		Reference:     w.NextRef(),
		Action:        manage.ActionCreate,
		Name:          name,
		ParentName:    parent,
	})
	return err
}

// genericActionCommand builds a Command for the "I <action> ..." steps that
// don't carry action-specific details (a rename's new name, a move's
// destination) in their Gherkin text.
func genericActionCommand(w *acceptance.World, action, target string) manage.Command {
	cmd := manage.Command{
		CorrelationID: w.NextRef(),
		TargetName:    target,
	}
	switch action {
	case "rename":
		cmd.Action = manage.ActionRename
		cmd.Name = target + " Renamed"
	case "move":
		parent := "Garage"
		cmd.Action = manage.ActionMove
		cmd.ParentName = &parent
	case "archive":
		cmd.Action = manage.ActionArchive
	}
	return cmd
}

// ancestryDepth walks name's Parent chain up to the root, returning how many
// hops (inclusive of name itself) that took and the root ancestor's name.
func ancestryDepth(w *acceptance.World, name string) (depth int, root string, err error) {
	cur := name
	for {
		loc, err := w.Storage.FindByName(context.Background(), cur)
		if err != nil {
			return 0, "", fmt.Errorf("find %q: %w", cur, err)
		}
		depth++
		if loc.Parent == "" {
			return depth, cur, nil
		}
		cur = loc.Parent
	}
}

// assertNotCreated confirms name was not created by a rejected request,
// distinguishing "never existed" from "a pre-existing sibling collided and
// must be unchanged".
func assertNotCreated(w *acceptance.World, name string) error {
	if !w.SnapshotFound[name] {
		if _, err := w.Storage.FindByName(context.Background(), name); !errors.Is(err, entities.ErrLocationNotFound) {
			return fmt.Errorf("expected %q not to have been created", name)
		}
		return nil
	}
	return assertUnchanged(w, name)
}

// assertUnchanged confirms name's current state matches the snapshot taken
// before a (rejected) command was attempted against it.
func assertUnchanged(w *acceptance.World, name string) error {
	want, ok := w.SnapshotBefore[name]
	if !ok {
		return fmt.Errorf("no snapshot recorded for %q", name)
	}
	got, err := w.Storage.FindByName(context.Background(), name)
	if err != nil {
		return fmt.Errorf("find %q: %w", name, err)
	}
	if got != want {
		return fmt.Errorf("expected %q to be unchanged: got %+v, want %+v", name, got, want)
	}
	return nil
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
