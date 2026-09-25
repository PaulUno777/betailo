package ports

import "context"

// TxManager runs fn atomically: either every write fn makes commits
// together, or none do. The MongoDB implementation wraps a client
// session/transaction (see infrastructure/mongo/tx_manager.go);
// usecases depend only on this interface, never on the driver's session
// API — the Dependency Inversion Principle applied to the one place in
// this codebase that needs a real multi-document transaction.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
