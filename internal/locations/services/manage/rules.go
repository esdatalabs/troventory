package manage

import (
	"context"
	"errors"
	"fmt"

	"github.com/esdatalabs/troventory/internal/locations/entities"
)

// create adds a new location, optionally nested under a parent. It is
// idempotent by cmd.Reference: resubmitting the same reference is a
// successful no-op rather than a second creation.
func (s *Service) create(ctx context.Context, cmd Command) error {
	if cmd.Reference != "" {
		_, err := s.storage.FindByReference(ctx, cmd.Reference)
		switch {
		case err == nil:
			return nil // already created for this reference
		case errors.Is(err, entities.ErrLocationNotFound):
			// not created yet — fall through
		default:
			return fmt.Errorf("check idempotency reference %q: %w", cmd.Reference, err)
		}
	}

	parentName := parentNameOf(cmd.ParentName)
	if parentName != "" {
		parent, err := s.storage.FindByName(ctx, parentName)
		if err != nil {
			return fmt.Errorf("find parent %q: %w", parentName, err)
		}
		if parent.Archived {
			return fmt.Errorf("parent %q: %w", parentName, entities.ErrLocationArchived)
		}
	}

	if err := s.rejectDuplicateSibling(ctx, parentName, cmd.Name, ""); err != nil {
		return err
	}

	loc := entities.Location{
		Name:     cmd.Name,
		Parent:   parentName,
		Archived: false,
	}
	if err := s.storage.Save(ctx, loc, cmd.Reference); err != nil {
		return fmt.Errorf("save location %q: %w", cmd.Name, err)
	}
	return nil
}

// rename changes target's Name to cmd.Name, rejecting collisions with an
// existing sibling under the same parent.
func (s *Service) rename(ctx context.Context, cmd Command) error {
	target, err := s.findActive(ctx, cmd.TargetName)
	if err != nil {
		return err
	}

	if err := s.rejectDuplicateSibling(ctx, target.Parent, cmd.Name, target.Name); err != nil {
		return err
	}

	target.Name = cmd.Name
	if err := s.storage.Save(ctx, target, ""); err != nil {
		return fmt.Errorf("save location %q: %w", cmd.Name, err)
	}
	return nil
}

// move reparents target under cmd.ParentName (nil meaning top-level),
// rejecting an archived destination and a destination that would make
// target its own descendant.
func (s *Service) move(ctx context.Context, cmd Command) error {
	target, err := s.findActive(ctx, cmd.TargetName)
	if err != nil {
		return err
	}

	destName := parentNameOf(cmd.ParentName)
	if destName != "" {
		dest, err := s.storage.FindByName(ctx, destName)
		if err != nil {
			return fmt.Errorf("find parent %q: %w", destName, err)
		}
		if dest.Archived {
			return fmt.Errorf("parent %q: %w", destName, entities.ErrLocationArchived)
		}

		cyclic, err := s.wouldCycle(ctx, target.Name, destName)
		if err != nil {
			return fmt.Errorf("check cyclic move of %q under %q: %w", target.Name, destName, err)
		}
		if cyclic {
			return fmt.Errorf("move %q under %q: %w", target.Name, destName, entities.ErrCyclicMove)
		}
	}

	target.Parent = destName
	if err := s.storage.Save(ctx, target, ""); err != nil {
		return fmt.Errorf("save location %q: %w", target.Name, err)
	}
	return nil
}

// archive soft-removes target, rejecting the operation if it still has
// non-archived children.
func (s *Service) archive(ctx context.Context, cmd Command) error {
	target, err := s.findActive(ctx, cmd.TargetName)
	if err != nil {
		return err
	}

	children, err := s.storage.ChildrenOf(ctx, target.Name)
	if err != nil {
		return fmt.Errorf("list children of %q: %w", target.Name, err)
	}
	for _, child := range children {
		if !child.Archived {
			return fmt.Errorf("location %q: %w", target.Name, entities.ErrLocationHasActiveChildren)
		}
	}

	target.Archived = true
	if err := s.storage.Save(ctx, target, ""); err != nil {
		return fmt.Errorf("save location %q: %w", target.Name, err)
	}
	return nil
}

// findActive looks up name, rejecting a location that does not exist or has
// already been archived — the shared precondition for rename, move, and
// archive.
func (s *Service) findActive(ctx context.Context, name string) (entities.Location, error) {
	loc, err := s.storage.FindByName(ctx, name)
	if err != nil {
		return entities.Location{}, fmt.Errorf("find location %q: %w", name, err)
	}
	if loc.Archived {
		return entities.Location{}, fmt.Errorf("location %q: %w", name, entities.ErrLocationArchived)
	}
	return loc, nil
}

// rejectDuplicateSibling returns entities.ErrDuplicateLocationName if a
// location other than excludeName already exists under parentName with the
// given name.
func (s *Service) rejectDuplicateSibling(ctx context.Context, parentName, name, excludeName string) error {
	siblings, err := s.storage.ChildrenOf(ctx, parentName)
	if err != nil {
		return fmt.Errorf("list children of %q: %w", parentName, err)
	}
	for _, sib := range siblings {
		if sib.Name != excludeName && sib.Name == name {
			return fmt.Errorf("location %q under %q: %w", name, parentName, entities.ErrDuplicateLocationName)
		}
	}
	return nil
}

// wouldCycle reports whether moving targetName under destName would make
// targetName its own descendant (including moving it under itself), by
// walking destName's ancestor chain up to the top level.
func (s *Service) wouldCycle(ctx context.Context, targetName, destName string) (bool, error) {
	cur := destName
	for cur != "" {
		if cur == targetName {
			return true, nil
		}
		loc, err := s.storage.FindByName(ctx, cur)
		if err != nil {
			return false, fmt.Errorf("find location %q: %w", cur, err)
		}
		cur = loc.Parent
	}
	return false, nil
}

// parentNameOf normalizes a Command's ParentName pointer to a plain name,
// where "" means top-level.
func parentNameOf(parent *string) string {
	if parent == nil {
		return ""
	}
	return *parent
}
