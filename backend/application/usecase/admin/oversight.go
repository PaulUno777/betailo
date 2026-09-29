package admin

import (
	"context"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListAllCycles backs the admin console's "Cycles & annonces" page —
// every cycle platform-wide, regardless of which farm owns it.
type ListAllCycles struct {
	Cycles ports.CycleStore
}

func (uc *ListAllCycles) Execute(ctx context.Context) ([]*domain.Cycle, error) {
	var out []*domain.Cycle
	for _, status := range []domain.CycleStatus{
		domain.CycleStatusPlanned, domain.CycleStatusInProgress, domain.CycleStatusDone, domain.CycleStatusCancelled,
	} {
		cs, err := uc.Cycles.ListByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		out = append(out, cs...)
	}
	return out, nil
}

// ListAllListings is the same page's listings half — every listing
// platform-wide, published or not, flagged or not (unlike the public
// marketplace's SearchListings, which deliberately excludes both).
type ListAllListings struct {
	Cycles   ports.CycleStore
	Listings ports.ListingStore
}

func (uc *ListAllListings) Execute(ctx context.Context) ([]*domain.Listing, error) {
	var out []*domain.Listing
	for _, status := range []domain.CycleStatus{
		domain.CycleStatusPlanned, domain.CycleStatusInProgress, domain.CycleStatusDone, domain.CycleStatusCancelled,
	} {
		cs, err := uc.Cycles.ListByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			l, err := uc.Listings.FindByCycleID(ctx, c.ID)
			if err != nil {
				continue // no listing published for this cycle yet
			}
			out = append(out, l)
		}
	}
	return out, nil
}
