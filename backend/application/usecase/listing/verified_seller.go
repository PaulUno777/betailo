package listing

import (
	"fmt"

	"github.com/pauluno777/betailo/backend/domain"
)

// errFarmNotApproved is the refusal for putting a listing live from a
// farm staff haven't approved. Anyone can open a farm and prepare
// cycles and draft listings in it; only an approved farm's listings
// reach the marketplace.
func errFarmNotApproved(f *domain.Farm) error {
	return domain.CodedWith("farm.not_approved", map[string]any{"status": string(f.Status)},
		fmt.Errorf("listing: its farm is %s, not approved: %w", f.Status, domain.ErrForbidden))
}
