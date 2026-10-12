package review

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/notify"
	"github.com/pauluno777/betailo/backend/domain"
)

// Reply posts the farm owner's one public answer to a review ("Réponse
// de l'éleveur"). It can't be edited afterwards, so the conversation a
// buyer reads is the one that happened.
type Reply struct {
	Store         ports.ReviewStore
	Farms         ports.FarmRepository
	Clock         ports.Clock
	Notifications ports.NotificationRepository
}

type ReplyInput struct {
	ReviewID bson.ObjectID
	OwnerID  bson.ObjectID
	Text     string
}

// Execute returns domain.ErrForbidden for anyone but the farm's owner,
// domain.ErrConflict if there's already a reply, domain.ErrInvalid for
// an empty or too-long text.
func (uc *Reply) Execute(ctx context.Context, in ReplyInput) (*domain.Review, error) {
	text, err := domain.ValidateReviewText(in.Text, domain.MaxReviewReplyLen)
	if err != nil {
		return nil, err
	}
	if text == "" {
		return nil, fmt.Errorf("review: reply text is required: %w", domain.ErrInvalid)
	}
	rv, err := uc.Store.FindByID(ctx, in.ReviewID)
	if err != nil {
		return nil, err
	}
	f, err := uc.Farms.FindByID(ctx, rv.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("review: only the farm's owner can reply: %w", domain.ErrForbidden)
	}
	reply := domain.ReviewReply{Text: text, At: uc.Clock.Now()}
	if err := uc.Store.SetReply(ctx, rv.ID, reply); err != nil {
		return nil, err
	}
	rv.Reply = &reply
	notify.Send(ctx, uc.Notifications, reply.At, notify.Notice{
		UserID: rv.AuthorID, Type: domain.NotifReviewReply, Params: map[string]string{"farm": f.Name},
		Link: domain.LinkBuyerReservations, Fallback: f.Name + " a répondu à votre avis.",
	})
	return rv, nil
}
