package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// TxManager implements ports.TxManager with a client session and a
// multi-document transaction (startSession/commitTransaction) — "the
// transaction multi-documents ... garantit que la décrémentation de
// capacité et la création de la réservation réussissent ou échouent
// ensemble" (architecture document).
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
