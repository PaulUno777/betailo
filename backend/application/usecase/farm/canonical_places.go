package farm

import (
	"context"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// farmStatuses is every review state, to visit every farm.
var farmStatuses = []domain.FarmStatus{domain.FarmDraft, domain.FarmPending, domain.FarmApproved, domain.FarmRejected, domain.FarmSuspended}

// CanonicalizePlaces rewrites every farm's region and town the way a
// farm saved today stores them: the catalog's keys for a listed region
// and town ("Ouest" → "ouest", "Bafoussam" → "bafoussam"), the text
// otherwise. Farms saved before regions and towns had keys get them;
// it is idempotent, so it runs on every startup. A place the catalog
// no longer accepts is left as it was. It returns how many farms
// changed.
func CanonicalizePlaces(ctx context.Context, farms ports.FarmRepository, src ports.CatalogSource) (int, error) {
	changed := 0
	for _, status := range farmStatuses {
		list, err := farms.ListByStatus(ctx, status)
		if err != nil {
			return changed, err
		}
		for _, f := range list {
			where, err := normalizePlace(ctx, src, f.Country, f.Region, f.Location)
			if err != nil {
				continue
			}
			if f.Region == where.Region && f.Location == where.Town && f.Country == where.Country.Code {
				continue
			}
			f.Country, f.Region, f.Location = where.Country.Code, where.Region, where.Town
			if err := farms.Update(ctx, f); err != nil {
				return changed, err
			}
			changed++
		}
	}
	return changed, nil
}
