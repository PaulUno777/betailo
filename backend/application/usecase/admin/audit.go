package admin

import (
	"context"
	"log/slog"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Journal writes the back office's audit trail. Every staff action
// that changes something goes through Record once it succeeded, and so
// does every look at an identity document.
type Journal struct {
	Log   ports.AuditLog
	Users ports.UserRepository
	Clock ports.Clock
	// Logger hears about a line that couldn't be written: the action
	// itself already happened, so the failure can't be returned to the
	// person who did it, but it must not go unnoticed.
	Logger *slog.Logger
}

// Entry is what the caller knows about the action.
type Entry struct {
	Action      string
	TargetType  string
	TargetID    bson.ObjectID
	TargetLabel string
	Detail      string
}

// Record never fails the request it documents.
func (j *Journal) Record(ctx context.Context, actor Actor, e Entry) {
	if j == nil || j.Log == nil {
		return
	}
	name := ""
	if u, err := j.Users.FindByID(ctx, actor.ID); err == nil {
		name = displayName(u)
	}
	detail := strings.TrimSpace(e.Detail)
	if r := []rune(detail); len(r) > domain.MaxAuditDetailLen {
		detail = string(r[:domain.MaxAuditDetailLen])
	}
	err := j.Log.Record(ctx, &domain.AdminAction{
		ActorID: actor.ID, ActorName: name, ActorRole: actor.Role, Action: e.Action,
		TargetType: e.TargetType, TargetID: e.TargetID, TargetLabel: e.TargetLabel, Detail: detail, CreatedAt: j.Clock.Now(),
	})
	if err != nil && j.Logger != nil {
		j.Logger.Error("audit trail: action not recorded", slog.String("action", e.Action), slog.String("actor", actor.ID.Hex()), slog.Any("err", err))
	}
}

// DefaultAuditPage is how many lines the journal page shows.
const DefaultAuditPage = 200

// ListAudit backs the "Journal" page: latest actions first.
type ListAudit struct {
	Log ports.AuditLog
}

func (uc *ListAudit) Execute(ctx context.Context) ([]*domain.AdminAction, error) {
	return uc.Log.List(ctx, DefaultAuditPage)
}
