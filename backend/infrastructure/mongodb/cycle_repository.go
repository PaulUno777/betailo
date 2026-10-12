package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// healthEventDoc mirrors domain.HealthEvent for BSON persistence.
type healthEventDoc struct {
	Type        string    `bson:"type"`
	Description string    `bson:"description"`
	Date        time.Time `bson:"date"`
}

// cycleDoc mirrors domain.Cycle for BSON persistence. domain.Cycle
// itself carries no bson tags (see domain/cycle.go); mapping happens
// only here, at the infrastructure boundary.
type cycleDoc struct {
	ID                bson.ObjectID      `bson:"_id,omitempty"`
	Reference         string             `bson:"reference,omitempty"`
	FarmID            bson.ObjectID      `bson:"farmId"`
	Species           string             `bson:"species"`
	Type              string             `bson:"type"`
	Status            domain.CycleStatus `bson:"status"`
	Capacity          int                `bson:"capacity"`
	AvailableQuantity int                `bson:"availableQuantity"`
	StartsAt          time.Time          `bson:"startsAt"`
	EndsAt            *time.Time         `bson:"endsAt,omitempty"`
	MinPerReservation int                `bson:"minPerReservation,omitempty"`
	MaxPerReservation int                `bson:"maxPerReservation,omitempty"`
	HealthLog         []healthEventDoc   `bson:"healthLog,omitempty"`
}

func fromDomainCycle(c *domain.Cycle) cycleDoc {
	doc := cycleDoc{
		ID:                c.ID,
		Reference:         c.Reference,
		FarmID:            c.FarmID,
		Species:           c.Species,
		Type:              c.Type,
		Status:            c.Status,
		Capacity:          c.Capacity,
		AvailableQuantity: c.AvailableQuantity,
		StartsAt:          c.StartsAt,
		EndsAt:            c.EndsAt,
		MinPerReservation: c.MinPerReservation,
		MaxPerReservation: c.MaxPerReservation,
	}
	for _, e := range c.HealthLog {
		doc.HealthLog = append(doc.HealthLog, healthEventDoc{Type: e.Type, Description: e.Description, Date: e.Date})
	}
	return doc
}

func (d cycleDoc) toDomain() *domain.Cycle {
	c := &domain.Cycle{
		ID:                d.ID,
		Reference:         d.Reference,
		FarmID:            d.FarmID,
		Species:           d.Species,
		Type:              d.Type,
		Status:            d.Status,
		Capacity:          d.Capacity,
		AvailableQuantity: d.AvailableQuantity,
		StartsAt:          d.StartsAt,
		EndsAt:            d.EndsAt,
		MinPerReservation: d.MinPerReservation,
		MaxPerReservation: d.MaxPerReservation,
	}
	for _, e := range d.HealthLog {
		c.HealthLog = append(c.HealthLog, domain.HealthEvent{Type: e.Type, Description: e.Description, Date: e.Date})
	}
	return c
}

// CycleRepository implements ports.CycleRepository (reservation's
// narrow slice) and ports.CycleStore (the fuller CRUD port
// application/usecase/cycle depends on) against the "cycles"
// collection — see ports.CycleStore's doc comment for why one concrete
// type implements two interfaces here.
type CycleRepository struct {
	Coll *mongo.Collection
}

var (
	_ ports.CycleRepository = (*CycleRepository)(nil)
	_ ports.CycleStore      = (*CycleRepository)(nil)
)

// Create implements ports.CycleStore.
func (r *CycleRepository) Create(ctx context.Context, c *domain.Cycle) error {
	if c.ID.IsZero() {
		c.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainCycle(c))
	return err
}

// FindByID implements ports.CycleStore.
func (r *CycleRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	var doc cycleDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// ListByFarm implements ports.CycleStore.
func (r *CycleRepository) ListByFarm(ctx context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"farmId": farmID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Cycle
	for cur.Next(ctx) {
		var doc cycleDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// ListByStatus implements ports.CycleStore.
func (r *CycleRepository) ListByStatus(ctx context.Context, status domain.CycleStatus) ([]*domain.Cycle, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"status": status})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Cycle
	for cur.Next(ctx) {
		var doc cycleDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// UpdateStatus implements ports.CycleStore.
func (r *CycleRepository) UpdateStatus(ctx context.Context, id bson.ObjectID, status domain.CycleStatus) error {
	res, err := r.Coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": status}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// AppendHealthEvent implements ports.CycleStore. The filter only
// matches while index maxEntries-1 doesn't exist yet, which caps the
// log in the same atomic update that grows it.
func (r *CycleRepository) AppendHealthEvent(ctx context.Context, id bson.ObjectID, e domain.HealthEvent, maxEntries int) error {
	full := fmt.Sprintf("healthLog.%d", maxEntries-1)
	res, err := r.Coll.UpdateOne(ctx,
		bson.M{"_id": id, full: bson.M{"$exists": false}},
		bson.M{"$push": bson.M{"healthLog": healthEventDoc{Type: e.Type, Description: e.Description, Date: e.Date}}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		// Either the cycle doesn't exist or its log is full.
		if n, err := r.Coll.CountDocuments(ctx, bson.M{"_id": id}); err != nil {
			return err
		} else if n == 0 {
			return domain.ErrNotFound
		}
		return fmt.Errorf("cycle: health log is full: %w", domain.ErrConflict)
	}
	return nil
}

// UpdateDetails implements ports.CycleEditor.
func (r *CycleRepository) UpdateDetails(ctx context.Context, id bson.ObjectID, d ports.CycleDetails, capacityDelta int) error {
	set := bson.M{"startsAt": d.StartsAt, "minPerReservation": d.MinPerReservation, "maxPerReservation": d.MaxPerReservation}
	update := bson.M{"$set": set, "$inc": bson.M{"capacity": capacityDelta, "availableQuantity": capacityDelta}}
	if d.EndsAt != nil {
		set["endsAt"] = *d.EndsAt
	} else {
		update["$unset"] = bson.M{"endsAt": ""}
	}
	filter := bson.M{"_id": id}
	if capacityDelta < 0 {
		filter["availableQuantity"] = bson.M{"$gte": -capacityDelta}
	}
	res, err := r.Coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		if n, err := r.Coll.CountDocuments(ctx, bson.M{"_id": id}); err != nil {
			return err
		} else if n == 0 {
			return domain.ErrNotFound
		}
		return fmt.Errorf("cycle: can't drop capacity below the places already reserved: %w", domain.ErrConflict)
	}
	return nil
}

// Delete implements ports.CycleStore.
func (r *CycleRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.Coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DecrementAvailability implements ports.CycleRepository via the atomic
// findOneAndUpdate + $inc described in the sequence diagram: the
// condition (availableQuantity >= quantity) and the decrement happen in
// a single document-level operation, which MongoDB guarantees is
// atomic — no explicit lock needed even under concurrent reservations.
func (r *CycleRepository) DecrementAvailability(ctx context.Context, cycleID bson.ObjectID, quantity int) error {
	filter := bson.M{"_id": cycleID, "availableQuantity": bson.M{"$gte": quantity}}
	update := bson.M{"$inc": bson.M{"availableQuantity": -quantity}}

	if err := r.Coll.FindOneAndUpdate(ctx, filter, update).Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.ErrInsufficientCapacity
		}
		return err
	}
	return nil
}

// IncrementAvailability implements ports.CycleRepository — the inverse
// of DecrementAvailability, used to restore capacity when a reservation
// that had claimed it is cancelled (an unpaid one expiring past its
// deadline, or an explicit cancellation). No capacity condition needed
// here: giving quantity back can never push availableQuantity past the
// cycle's own Capacity as long as callers only restore what they
// previously decremented — enforced by the usecase, not this method.
func (r *CycleRepository) IncrementAvailability(ctx context.Context, cycleID bson.ObjectID, quantity int) error {
	filter := bson.M{"_id": cycleID}
	update := bson.M{"$inc": bson.M{"availableQuantity": quantity}}

	res, err := r.Coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}
