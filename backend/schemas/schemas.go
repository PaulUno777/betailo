// Package schemas embeds one MongoDB $jsonSchema validator per
// collection, written by hand rather than generated from the Go
// structs so the two can be reviewed independently (a struct change
// doesn't silently loosen validation). infrastructure/mongodb's
// Bootstrap loads these at startup and attaches each one to its
// collection — see the data model section of the "Concepts &
// Architecture" document for the schema these mirror.
package schemas

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed *.json
var files embed.FS

// Names lists every collection with a validator, in the order
// bootstrap creates them (dependency order: a collection referenced by
// another comes first, though Mongo itself does not enforce it).
var Names = []string{
	"users",
	"refresh_tokens",
	"farms",
	"cycles",
	"listings",
	"reservations",
	"reviews",
	"reports",
	"forum_posts",
	"notifications",
	"favorites",
	"counters",
	"admin_actions",
}

// Load parses the named collection's $jsonSchema validator document.
// It returns an error rather than panicking so a bad embed (typo in a
// file name, invalid JSON) surfaces as a startup failure instead of a
// silent skip.
func Load(collection string) (map[string]any, error) {
	raw, err := files.ReadFile(collection + ".json")
	if err != nil {
		return nil, fmt.Errorf("schemas: read %s.json: %w", collection, err)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("schemas: parse %s.json: %w", collection, err)
	}

	return schema, nil
}
