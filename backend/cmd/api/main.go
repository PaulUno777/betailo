// Command api is the single entry point of the Betailo backend.
//
// The MVP ships as one binary (see the "Concepts & Architecture"
// document, section "Vue d'ensemble de l'architecture") rather than a
// set of microservices: simpler to run, deploy and reason about at this
// stage. All the wiring lives in internal/app; this file only calls it.
package main

import (
	"context"
	"log"

	"github.com/pauluno777/betailo/backend/internal/app"
)

func main() {
	ctx := context.Background()

	application, cleanup, err := app.Initialize(ctx)
	if err != nil {
		log.Fatalf("app: %v", err)
	}
	defer cleanup()

	if err := application.Bootstrap(ctx); err != nil {
		log.Fatalf("bootstrap: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}
