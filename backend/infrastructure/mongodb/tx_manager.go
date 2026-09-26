package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// TxManager implements ports.TxManager with a client session and a
// multi-document transaction (startSession/commitTransaction): the
// capacity decrement and the reservation insert succeed or fail
// together (see the architecture document's transaction section).
type TxManager struct {
	Client *mongo.Client
}

var _ ports.TxManager = (*TxManager)(nil)

// WithinTx implements ports.TxManager.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := m.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	})
	return err
}
