package httpserver

import (
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/dashboard"
)

type cycleSummaryResponse struct {
	Cycle             cycleResponse `json:"cycle"`
	FarmName          string        `json:"farmName,omitempty"`
	FillRate          float64       `json:"fillRate"`
	NextMilestone     string        `json:"nextMilestone,omitempty"`
	NextMilestoneDate time.Time     `json:"nextMilestoneDate,omitzero"`
}

type sellerDashboardResponse struct {
	TrustScore             float64 `json:"trustScore"`
	ReservationsThisPeriod int     `json:"reservationsThisPeriod"`
	// ReservationsPreviousPeriod is last month's count, to compare with.
	ReservationsPreviousPeriod int     `json:"reservationsPreviousPeriod"`
	EscrowHeldTotal            float64 `json:"escrowHeldTotal"`
	HeldReservations           int     `json:"heldReservations"`
	ReleasedThisPeriod         float64 `json:"releasedThisPeriod"`
	// NewReservations were made in the last seven days.
	NewReservations int `json:"newReservations"`
	// FarmCount is how many farms the figures cover: one, or all of
	// the account's on the combined view.
	FarmCount               int                    `json:"farmCount"`
	ReviewAverage           float64                `json:"reviewAverage"`
	ReviewCount             int                    `json:"reviewCount"`
	WeeklyReservationCounts [8]int                 `json:"weeklyReservationCounts"`
	ActiveCycles            []cycleSummaryResponse `json:"activeCycles"`
	LatestReservations      []dashboardReservation `json:"latestReservations"`
	Todos                   []todoResponse         `json:"todos"`
}

// dashboardReservation is a reservation on the seller's dashboard,
// with who made it and on which cycle. BuyerName is the buyer's public
// display name, empty when they never set one.
type dashboardReservation struct {
	reservationResponse
	BuyerName string `json:"buyerName,omitempty"`
	CycleRef  string `json:"cycleRef,omitempty"`
	Species   string `json:"species,omitempty"`
}

type todoResponse struct {
	Kind     string `json:"kind"`
	Count    int    `json:"count,omitempty"`
	CycleID  string `json:"cycleId,omitempty"`
	CycleRef string `json:"cycleRef,omitempty"`
}

// handleSellerDashboard serves one farm's dashboard, or, when the route
// has no farm (`GET /me/seller-dashboard`), every farm of the caller
// summed: the design's "Toutes mes fermes".
func handleSellerDashboard(uc *dashboard.SellerDashboard, combined bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		in := dashboard.SellerDashboardInput{OwnerID: claims.UserID}
		if !combined {
			farmID, ok := pathObjectID(w, r, "farmId")
			if !ok {
				return
			}
			in.FarmID = farmID
		}

		out, err := uc.Execute(r.Context(), in)
		if err != nil {
			writeError(w, err)
			return
		}

		cycles := make([]cycleSummaryResponse, 0, len(out.ActiveCycles))
		for _, c := range out.ActiveCycles {
			cycles = append(cycles, cycleSummaryResponse{
				Cycle: toCycleResponse(c.Cycle), FarmName: c.FarmName, FillRate: c.FillRate,
				NextMilestone: c.NextMilestone, NextMilestoneDate: c.NextMilestoneDate,
			})
		}
		latest := make([]dashboardReservation, 0, len(out.LatestReservations))
		for _, res := range out.LatestReservations {
			item := dashboardReservation{reservationResponse: toReservationResponse(res), BuyerName: out.LatestBuyers[res.ID]}
			if c := out.LatestCycles[res.ID]; c != nil {
				item.CycleRef, item.Species = c.Reference, c.Species
			}
			latest = append(latest, item)
		}

		todos := make([]todoResponse, 0, len(out.Todos))
		for _, td := range out.Todos {
			t := todoResponse{Kind: td.Kind, Count: td.Count, CycleRef: td.CycleRef}
			if !td.CycleID.IsZero() {
				t.CycleID = td.CycleID.Hex()
			}
			todos = append(todos, t)
		}

		writeJSON(w, http.StatusOK, sellerDashboardResponse{
			TrustScore: out.TrustScore, ReservationsThisPeriod: out.ReservationsThisPeriod,
			ReservationsPreviousPeriod: out.ReservationsPreviousPeriod,
			EscrowHeldTotal:            out.EscrowHeldTotal, HeldReservations: out.HeldReservations, ReleasedThisPeriod: out.ReleasedThisPeriod,
			NewReservations: out.NewReservations, FarmCount: out.FarmCount, ReviewAverage: out.ReviewAverage, ReviewCount: out.ReviewCount,
			WeeklyReservationCounts: out.WeeklyReservationCounts, ActiveCycles: cycles, LatestReservations: latest,
			Todos: todos,
		})
	}
}
