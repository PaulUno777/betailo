package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

// catalogVersionResponse is one line of a reference's history.
type catalogVersionResponse struct {
	Name        string    `json:"name"`
	Version     int       `json:"version"`
	Note        string    `json:"note,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
	// FromSeed is a version the platform shipped with, not staff's.
	FromSeed bool `json:"fromSeed"`
}

func toVersionResponse(p *domain.CatalogPart) catalogVersionResponse {
	return catalogVersionResponse{Name: p.Name, Version: p.Version, Note: p.Note, PublishedAt: p.PublishedAt, FromSeed: p.PublishedBy.IsZero()}
}

// handleAdminCatalogVersions lists published versions, newest first —
// of one reference (?name=places) or of all of them
// (GET /admin/reference/versions).
func handleAdminCatalogVersions(repo ports.CatalogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name != "" && !domain.IsPart(name) {
			writeError(w, domain.Coded("catalog.part_unknown", domain.ErrNotFound))
			return
		}
		parts, err := repo.List(r.Context(), name)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]catalogVersionResponse, 0, len(parts))
		for _, p := range parts {
			out = append(out, toVersionResponse(p))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// handleAdminCatalogSchema serves one reference's JSON Schema
// (GET /admin/reference/schema/{name}?lang=fr|en): the editor checks,
// completes and explains with the API's own rules.
func handleAdminCatalogSchema() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")
		if lang != "en" {
			lang = "fr"
		}
		schema := domain.CatalogPartSchema(r.PathValue("name"), lang)
		if schema == nil {
			writeError(w, domain.Coded("catalog.part_unknown", domain.ErrNotFound))
			return
		}
		writeJSON(w, http.StatusOK, schema)
	}
}

// handleAdminCatalogVersion serves one published version of a reference
// (GET /admin/reference/versions/{name}/{version}), to compare or restore.
func handleAdminCatalogVersion(repo ports.CatalogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version, err := strconv.Atoi(r.PathValue("version"))
		if err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		p, err := repo.Get(r.Context(), r.PathValue("name"), version)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, struct {
			catalogVersionResponse
			Content json.RawMessage `json:"content"`
		}{toVersionResponse(p), p.Content})
	}
}

// handleAdminCatalogUsage counts what uses each reference entry
// (GET /admin/reference/usage), keyed like "species:poussins".
func handleAdminCatalogUsage(usage ports.CatalogUsage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		counts, err := usage.Usage(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, counts)
	}
}

type changeAuthorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type changeErrorResponse struct {
	Code   string         `json:"code"`
	Params map[string]any `json:"params,omitempty"`
}

type changeResponse struct {
	ID string           `json:"id"`
	Op domain.CatalogOp `json:"op"`
	// Before is, for an update, the values it replaces.
	Before json.RawMessage      `json:"before,omitempty"`
	By     changeAuthorResponse `json:"by"`
	At     time.Time            `json:"at"`
	Error  *changeErrorResponse `json:"error,omitempty"`
}

// changesetResponse is the shared changeset replayed onto the version
// in force: its changes in order, the draft they make, and whether it
// can be published.
type changesetResponse struct {
	Rev int `json:"rev"`
	// BaseVersions is the version in force of each reference; Touched,
	// the references the changes act on (a publish versions those).
	BaseVersions map[string]int    `json:"baseVersions"`
	Touched      []string          `json:"touched"`
	Ready        bool              `json:"ready"`
	Problems     []string          `json:"problems"`
	Changes      []changeResponse  `json:"changes"`
	Draft        referenceResponse `json:"draft"`
}

func toChangesetResponse(ctx context.Context, users ports.UserRepository, v *catalog.ChangesetView) changesetResponse {
	names := map[bson.ObjectID]string{}
	name := func(id bson.ObjectID) string {
		if n, ok := names[id]; ok {
			return n
		}
		n := ""
		if users != nil {
			if u, err := users.FindByID(ctx, id); err == nil {
				n = u.DisplayName
				if n == "" {
					n = u.Email
				}
			}
		}
		names[id] = n
		return n
	}
	out := changesetResponse{
		Rev: v.Rev, BaseVersions: v.Base.Versions, Touched: v.Touched(), Ready: v.Ready(), Problems: v.Problems,
		Changes: make([]changeResponse, 0, len(v.Changes)),
		Draft:   toReferenceResponse(v.Draft),
	}
	if out.Touched == nil {
		out.Touched = []string{}
	}
	if out.Problems == nil {
		out.Problems = []string{}
	}
	for _, c := range v.Changes {
		cr := changeResponse{ID: c.ID.Hex(), Op: c.Op, Before: c.Before, By: changeAuthorResponse{ID: c.By.Hex(), Name: name(c.By)}, At: c.At}
		if c.Err != nil {
			code, params := domain.CodeOf(c.Err)
			cr.Error = &changeErrorResponse{Code: code, Params: params}
		}
		out.Changes = append(out.Changes, cr)
	}
	return out
}

// changesetHandler runs one changeset action for the staff member and
// answers with the changeset as it then stands.
func changesetHandler[Req any](journal *admin.Journal, run func(r *http.Request, actor admin.Actor, req Req) (*catalog.ChangesetView, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		var req Req
		if r.Body != nil && r.ContentLength != 0 && !readJSON(w, r, &req) {
			return
		}
		v, err := run(r, actor, req)
		if err != nil {
			writeError(w, err)
			return
		}
		var users ports.UserRepository
		if journal != nil {
			users = journal.Users
		}
		writeJSON(w, http.StatusOK, toChangesetResponse(r.Context(), users, v))
	}
}

type revRequest struct {
	Rev int `json:"rev"`
}

type queueRequest struct {
	Rev int                `json:"rev"`
	Ops []domain.CatalogOp `json:"ops"`
}

type importRequest struct {
	Rev     int             `json:"rev"`
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"`
}

type restoreRequest struct {
	Rev     int    `json:"rev"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type publishChangesetRequest struct {
	Rev  int    `json:"rev"`
	Note string `json:"note"`
}

func registerReferenceRoutes(mux *http.ServeMux, can func(domain.Permission, http.Handler) http.Handler, uc *catalog.Changeset, versions ports.CatalogRepository, usage ports.CatalogUsage, journal *admin.Journal) {
	manage := func(h http.Handler) http.Handler { return can(domain.PermReferenceManage, h) }
	mux.Handle("GET /admin/reference/schema/{name}", manage(handleAdminCatalogSchema()))
	mux.Handle("GET /admin/reference/versions", manage(handleAdminCatalogVersions(versions)))
	mux.Handle("GET /admin/reference/versions/{name}/{version}", manage(handleAdminCatalogVersion(versions)))
	mux.Handle("GET /admin/reference/usage", manage(handleAdminCatalogUsage(usage)))

	mux.Handle("GET /admin/reference/changeset", manage(changesetHandler(journal, func(r *http.Request, _ admin.Actor, _ struct{}) (*catalog.ChangesetView, error) {
		return uc.View(r.Context())
	})))
	mux.Handle("POST /admin/reference/changeset/ops", manage(changesetHandler(journal, func(r *http.Request, a admin.Actor, req queueRequest) (*catalog.ChangesetView, error) {
		return uc.Queue(r.Context(), req.Rev, req.Ops, a.ID)
	})))
	mux.Handle("DELETE /admin/reference/changeset/ops/{id}", manage(changesetHandler(journal, func(r *http.Request, _ admin.Actor, req revRequest) (*catalog.ChangesetView, error) {
		id, err := bson.ObjectIDFromHex(r.PathValue("id"))
		if err != nil {
			return nil, errBadRequestBody
		}
		return uc.Undo(r.Context(), revParam(r, req.Rev), id)
	})))
	mux.Handle("DELETE /admin/reference/changeset", manage(changesetHandler(journal, func(r *http.Request, _ admin.Actor, req revRequest) (*catalog.ChangesetView, error) {
		return uc.Clear(r.Context(), revParam(r, req.Rev))
	})))
	mux.Handle("POST /admin/reference/changeset/import", manage(changesetHandler(journal, func(r *http.Request, a admin.Actor, req importRequest) (*catalog.ChangesetView, error) {
		return uc.Import(r.Context(), req.Rev, req.Name, req.Content, a.ID)
	})))
	mux.Handle("POST /admin/reference/changeset/restore", manage(changesetHandler(journal, func(r *http.Request, a admin.Actor, req restoreRequest) (*catalog.ChangesetView, error) {
		return uc.Restore(r.Context(), req.Rev, req.Name, req.Version, a.ID)
	})))
	mux.Handle("POST /admin/reference/changeset/publish", manage(handleAdminPublishChangeset(uc, journal)))
}

// revParam is the changeset revision a DELETE is based on: its body's,
// or the ?rev= query parameter.
func revParam(r *http.Request, body int) int {
	if v, err := strconv.Atoi(r.URL.Query().Get("rev")); err == nil {
		return v
	}
	return body
}

// handleAdminPublishChangeset publishes the pending changes — a new
// version of each reference they touch — (POST
// /admin/reference/changeset/publish) and answers with the reference
// data now in force and the versions made.
func handleAdminPublishChangeset(uc *catalog.Changeset, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		var req publishChangesetRequest
		if !readJSON(w, r, &req) {
			return
		}
		published, n, err := uc.PublishAll(r.Context(), req.Rev, req.Note, actor.ID)
		if err != nil {
			writeError(w, err)
			return
		}
		made := make([]catalogVersionResponse, 0, len(published.Parts))
		labels := make([]string, 0, len(published.Parts))
		for _, p := range published.Parts {
			made = append(made, toVersionResponse(p))
			labels = append(labels, p.Name+" v"+strconv.Itoa(p.Version))
		}
		journal.Record(r.Context(), actor, admin.Entry{
			Action: domain.ActionCatalogPublish, TargetType: "catalog",
			TargetLabel: strings.Join(labels, ", "), Detail: strconv.Itoa(n) + " · " + strings.TrimSpace(req.Note),
		})
		writeJSON(w, http.StatusCreated, struct {
			Reference referenceResponse        `json:"reference"`
			Published []catalogVersionResponse `json:"published"`
		}{toReferenceResponse(published.Catalog), made})
	}
}
