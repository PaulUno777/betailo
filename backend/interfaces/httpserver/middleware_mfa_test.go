package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// A staff session that did not pass a second factor is kept out of the
// back office when the platform requires one, whatever its role.
func TestRequirePermission_SecondFactor(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	call := func(mfaRequired, sessionMFA bool) *httptest.ResponseRecorder {
		reached = false
		claims := ports.TokenClaims{UserID: bson.NewObjectID(), StaffRole: domain.StaffSuperAdmin, MFA: sessionMFA}
		req := httptest.NewRequest(http.MethodGet, "/admin/overview", nil)
		req = req.WithContext(context.WithValue(req.Context(), claimsContextKey{}, claims))
		rec := httptest.NewRecorder()
		RequirePermission(nil, domain.PermOverviewRead, mfaRequired)(next).ServeHTTP(rec, req)
		return rec
	}

	if rec := call(true, false); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "admin.mfa_required") || reached {
		t.Errorf("required, session without a factor = %d %s, reached %v; want 403 admin.mfa_required", rec.Code, rec.Body.String(), reached)
	}
	if rec := call(true, true); rec.Code != http.StatusOK || !reached {
		t.Errorf("required, verified session = %d, reached %v; want through", rec.Code, reached)
	}
	if rec := call(false, false); rec.Code != http.StatusOK || !reached {
		t.Errorf("not required = %d, reached %v; want through", rec.Code, reached)
	}
}
