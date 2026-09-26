package httpserver

import (
	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
)

// Dependencies is everything NewRouter needs to wire routes to
// usecases. A struct, not a growing parameter list, so adding the next
// business module's usecase (see the roadmap) doesn't change every
// existing call to NewRouter.
type Dependencies struct {
	Register *auth.Register
	Login    *auth.Login
	Signer   ports.TokenSigner
}
