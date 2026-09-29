package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/pauluno777/betailo/backend/application/usecase/report"
	"github.com/pauluno777/betailo/backend/domain"
)

type reportResponse struct {
	ID         string `json:"id"`
	TargetID   string `json:"targetId"`
	TargetType string `json:"targetType"`
	Reason     string `json:"reason"`
	Status     string `json:"status"`
}

func toReportResponse(r *domain.Report) reportResponse {
	return reportResponse{
		ID: r.ID.Hex(), TargetID: r.TargetID.Hex(), TargetType: string(r.TargetType),
		Reason: r.Reason, Status: string(r.Status),
	}
}

type createReportRequest struct {
	TargetID   string `json:"targetId"`
	TargetType string `json:"targetType"`
	Reason     string `json:"reason"`
}

func handleCreateReport(uc *report.CreateReport) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req createReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		targetID, err := parseObjectIDField(req.TargetID)
		if err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		rp, err := uc.Execute(r.Context(), report.CreateReportInput{
			ReporterID: claims.UserID, TargetID: targetID,
			TargetType: domain.ReportTargetType(req.TargetType), Reason: req.Reason,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, toReportResponse(rp))
	}
}
