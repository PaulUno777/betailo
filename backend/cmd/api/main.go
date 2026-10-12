// Command api is the single entry point of the Betailo backend.
//
// The MVP ships as one binary (see the "Concepts & Architecture"
// document, section "Vue d'ensemble de l'architecture") rather than a
// set of microservices: simpler to run, deploy and reason about at this
// stage. All the wiring lives in internal/app; this file only calls it.
package main

import (
	"context"
	"log/slog"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/internal/app"
)

func main() {
	ctx := context.Background()

	application, cleanup, err := app.Initialize(ctx)
	if err != nil {
		fail(slog.Default(), "app", err)
	}
	defer cleanup()

	if err := application.Bootstrap(ctx); err != nil {
		fail(application.Logger, "bootstrap", err)
	}

	if err := application.Run(); err != nil {
		fail(application.Logger, "run", err)
	}
}

// fail logs why the API can't start at ERROR level — never the INFO the
// standard logger would use once slog is the default — and exits, so the
// container restarts and its health check stays red until it's fixed.
func fail(logger *slog.Logger, stage string, err error) {
	attrs := []any{slog.String("stage", stage), slog.Any("err", err)}
	if mongo.IsDuplicateKeyError(err) {
		// Typically data written while the database had no indexes (it
		// was dropped and recreated under a running API).
		attrs = append(attrs, slog.String("hint", "a unique index can't be built over existing duplicate documents: remove the duplicates named above, then restart"))
	}
	logger.Error("api can't start", attrs...)
	os.Exit(1)
}
