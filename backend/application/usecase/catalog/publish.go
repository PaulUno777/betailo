package catalog

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// MaxNoteLen bounds a version's note ("what changed").
const MaxNoteLen = 300

// Publish makes the edited lists the next version of each reference
// they change — and only those — once they validate and keep every key
// already in use (an entry is deactivated, never removed). Versions are
// never edited in place, so what any listing was checked against stays
// readable.
type Publish struct {
	Repo  ports.CatalogRepository
	Clock ports.Clock
	// Cache, when set, forgets its copy so this instance serves the new
	// versions at once (others within their TTL).
	Cache *Cache
}

// PublishInput is the edited lists and the versions they are based on.
type PublishInput struct {
	// Base is the version of each reference the edit started from:
	// publishing over a newer one would silently undo someone's changes.
	Base  map[string]int
	Note  string
	Lists *domain.Catalog
	By    bson.ObjectID
}

// Published is the catalog now in force and the versions just made.
type Published struct {
	Catalog *domain.Catalog
	Parts   []*domain.CatalogPart
}

// Execute returns the new versions, or: catalog.no_changes; catalog.stale
// (ErrConflict) when a reference it changes was published since Base;
// catalog.note_invalid; catalog.invalid with the problems;
// catalog.key_removed with the keys dropped.
func (uc *Publish) Execute(ctx context.Context, in PublishInput) (*Published, error) {
	current, _, err := Current(ctx, uc.Repo)
	if err != nil {
		return nil, err
	}
	touched := domain.ChangedParts(current, in.Lists)
	if len(touched) == 0 {
		return nil, domain.Coded("catalog.no_changes", fmt.Errorf("catalog: nothing to publish: %w", domain.ErrInvalid))
	}
	for _, name := range touched {
		if in.Base[name] != current.Versions[name] {
			return nil, domain.CodedWith("catalog.stale", map[string]any{"reference": name, "version": current.Versions[name]},
				fmt.Errorf("catalog: %s edited from version %d, version %d is in force: %w", name, in.Base[name], current.Versions[name], domain.ErrConflict))
		}
	}
	note := strings.TrimSpace(in.Note)
	if note == "" || utf8.RuneCountInString(note) > MaxNoteLen {
		return nil, domain.CodedWith("catalog.note_invalid", map[string]any{"max": MaxNoteLen},
			fmt.Errorf("catalog: the note must be 1 to %d characters: %w", MaxNoteLen, domain.ErrInvalid))
	}
	if err := in.Lists.Validate(); err != nil {
		return nil, err
	}
	if removed := domain.CatalogRemovals(current, in.Lists); len(removed) > 0 {
		return nil, domain.CodedWith("catalog.key_removed", map[string]any{"keys": strings.Join(removed, ", ")},
			fmt.Errorf("catalog: %s would be removed; deactivate instead: %w", strings.Join(removed, ", "), domain.ErrInvalid))
	}

	now := uc.Clock.Now()
	next := in.Lists.Clone()
	next.Versions = map[string]int{}
	for k, v := range current.Versions {
		next.Versions[k] = v
	}
	out := &Published{Catalog: next}
	for _, name := range touched {
		content, err := in.Lists.PartContent(name)
		if err != nil {
			return nil, err
		}
		p := &domain.CatalogPart{Name: name, Version: current.Versions[name] + 1, Note: note, PublishedBy: in.By, PublishedAt: now, Content: content}
		if err := uc.Repo.Insert(ctx, p); err != nil {
			return nil, err
		}
		next.Versions[name] = p.Version
		out.Parts = append(out.Parts, p)
	}
	if uc.Cache != nil {
		uc.Cache.Forget()
	}
	return out, nil
}
