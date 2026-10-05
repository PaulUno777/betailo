// Package account is a user's own profile: what they see and edit on
// their account page.
package account

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// GetProfile returns the signed-in user.
type GetProfile struct {
	Users ports.UserRepository
}

func (uc *GetProfile) Execute(ctx context.Context, userID bson.ObjectID) (*domain.User, error) {
	return uc.Users.FindByID(ctx, userID)
}

// UpdateProfile edits the user's own display name, phone and city.
type UpdateProfile struct {
	Users ports.UserRepository
}

type UpdateProfileInput struct {
	UserID      bson.ObjectID
	DisplayName string
	Phone       string // as typed; stored E.164, empty clears it
	City        string
}

// Execute returns domain.ErrInvalid for an empty or too-long name, an
// invalid phone or a too-long city.
func (uc *UpdateProfile) Execute(ctx context.Context, in UpdateProfileInput) (*domain.User, error) {
	name := strings.TrimSpace(in.DisplayName)
	city := strings.TrimSpace(in.City)
	switch {
	case name == "":
		return nil, fmt.Errorf("account: a display name is required: %w", domain.ErrInvalid)
	case utf8.RuneCountInString(name) > domain.MaxDisplayNameLen:
		return nil, fmt.Errorf("account: display name must be at most %d characters: %w", domain.MaxDisplayNameLen, domain.ErrInvalid)
	case utf8.RuneCountInString(city) > domain.MaxCityLen:
		return nil, fmt.Errorf("account: city must be at most %d characters: %w", domain.MaxCityLen, domain.ErrInvalid)
	}
	phone := ""
	if strings.TrimSpace(in.Phone) != "" {
		p, err := domain.NormalizePhone(in.Phone)
		if err != nil {
			return nil, fmt.Errorf("account: phone: %w", err)
		}
		phone = p
	}

	u, err := uc.Users.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	u.DisplayName, u.Phone, u.City = name, phone, city
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
