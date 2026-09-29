package httpserver

import (
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/dashboard"
)

type cycleSummaryResponse struct {
	Cycle             cycleResponse `json:"cycle"`
	FillRate          float64       `json:"fillRate"`
	NextMilestone     string        `json:"nextMilestone,omitempty"`
	NextMilestoneDate time.Time     `json:"nextMilestoneDate,omitzero"`
}

type sellerDashboardResponse struct {
	TrustScore              float64                `json:"trustScore"`
	ReservationsThisPeriod  int                    `json:"reservationsThisPeriod"`
	EscrowHeldTotal         float64                `json:"escrowHeldTotal"`
	ReleasedThisPeriod      float64                `json:"releasedThisPeriod"`
	WeeklyReservationCounts [8]int                 `json:"weeklyReservationCounts"`
	ActiveCycles            []cycleSummaryResponse `json:"activeCycles"`
	LatestReservations      []reservationResponse  `json:"latestReservations"`
}

func handleSellerDashboard(uc *dashboard.SellerDashboard) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		farmID, ok := pathObjectID(w, r, "farmId")
		if !ok {
			return
		}

		out, err := uc.Execute(r.Context(), dashboard.SellerDashboardInput{FarmID: farmID, OwnerID: claims.UserID})
		if err != nil {
			writeError(w, err)
			return
		}

		cycles := make([]cycleSummaryResponse, 0, len(out.ActiveCycles))
		for _, c := range out.ActiveCycles {
			cycles = append(cycles, cycleSummaryResponse{
				Cycle: toCycleResponse(c.Cycle), FillRate: c.FillRate,
				NextMilestone: c.NextMilestone, NextMilestoneDate: c.NextMilestoneDate,
			})
		}
		latest := make([]reservationResponse, 0, len(out.LatestReservations))
		for _, res := range out.LatestReservations {
			latest = append(latest, toReservationResponse(res))
		}

		writeJSON(w, http.StatusOK, sellerDashboardResponse{
			TrustScore: out.TrustScore, ReservationsThisPeriod: out.ReservationsThisPeriod,
			EscrowHeldTotal: out.EscrowHeldTotal, ReleasedThisPeriod: out.ReleasedThisPeriod,
			WeeklyReservationCounts: out.WeeklyReservationCounts, ActiveCycles: cycles, LatestReservations: latest,
		})
	}
}
