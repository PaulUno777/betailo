// Package passkey is the WebAuthn factor, on go-webauthn.
package passkey

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Service implements ports.Passkeys for one relying party: the site's
// own host, and the origins its pages are served from.
type Service struct {
	web *webauthn.WebAuthn
}

var _ ports.Passkeys = (*Service)(nil)

// New builds the service. rpID is the site's host without scheme or
// port ("betailo.example", "localhost"); origins are the full origins a
// browser may present.
func New(displayName, rpID string, origins []string) (*Service, error) {
	web, err := webauthn.New(&webauthn.Config{RPDisplayName: displayName, RPID: rpID, RPOrigins: origins})
	if err != nil {
		return nil, fmt.Errorf("passkey: %w", err)
	}
	return &Service{web: web}, nil
}

// account adapts a domain.User to what the library asks of a user.
type account struct {
	user  *domain.User
	creds []webauthn.Credential
}

func (a account) WebAuthnID() []byte          { return a.user.ID[:] }
func (a account) WebAuthnName() string        { return a.user.Email }
func (a account) WebAuthnDisplayName() string { return a.user.Email }
func (a account) WebAuthnCredentials() []webauthn.Credential {
	return a.creds
}

func accountOf(u *domain.User) (account, error) {
	a := account{user: u}
	for _, p := range u.MFA.Passkeys {
		var c webauthn.Credential
		if err := json.Unmarshal(p.Data, &c); err != nil {
			return a, fmt.Errorf("passkey: stored credential: %w", err)
		}
		a.creds = append(a.creds, c)
	}
	return a, nil
}

// BeginRegistration implements ports.Passkeys. The account's existing
// passkeys are excluded, so the same authenticator isn't added twice.
func (s *Service) BeginRegistration(u *domain.User) ([]byte, []byte, error) {
	a, err := accountOf(u)
	if err != nil {
		return nil, nil, err
	}
	exclude := make([]protocol.CredentialDescriptor, 0, len(a.creds))
	for _, c := range a.creds {
		exclude = append(exclude, c.Descriptor())
	}
	creation, session, err := s.web.BeginRegistration(a, webauthn.WithExclusions(exclude))
	if err != nil {
		return nil, nil, err
	}
	return marshalBoth(creation, session)
}

// FinishRegistration implements ports.Passkeys.
func (s *Service) FinishRegistration(u *domain.User, session, response []byte) (*domain.Passkey, error) {
	a, err := accountOf(u)
	if err != nil {
		return nil, err
	}
	var sd webauthn.SessionData
	if err := json.Unmarshal(session, &sd); err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(response))
	if err != nil {
		return nil, err
	}
	cred, err := s.web.CreateCredential(a, sd, parsed)
	if err != nil {
		return nil, err
	}
	return toPasskey(cred)
}

// BeginLogin implements ports.Passkeys.
func (s *Service) BeginLogin(u *domain.User) ([]byte, []byte, error) {
	a, err := accountOf(u)
	if err != nil {
		return nil, nil, err
	}
	assertion, session, err := s.web.BeginLogin(a)
	if err != nil {
		return nil, nil, err
	}
	return marshalBoth(assertion, session)
}

// FinishLogin implements ports.Passkeys.
func (s *Service) FinishLogin(u *domain.User, session, response []byte) (*domain.Passkey, error) {
	a, err := accountOf(u)
	if err != nil {
		return nil, err
	}
	var sd webauthn.SessionData
	if err := json.Unmarshal(session, &sd); err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(response))
	if err != nil {
		return nil, err
	}
	cred, err := s.web.ValidateLogin(a, sd, parsed)
	if err != nil {
		return nil, err
	}
	if cred.Authenticator.CloneWarning {
		return nil, fmt.Errorf("passkey: signature counter went backwards, the credential may be cloned")
	}
	return toPasskey(cred)
}

func toPasskey(cred *webauthn.Credential) (*domain.Passkey, error) {
	data, err := json.Marshal(cred)
	if err != nil {
		return nil, err
	}
	return &domain.Passkey{ID: cred.ID, Data: data}, nil
}

func marshalBoth(options, session any) ([]byte, []byte, error) {
	o, err := json.Marshal(options)
	if err != nil {
		return nil, nil, err
	}
	sd, err := json.Marshal(session)
	if err != nil {
		return nil, nil, err
	}
	return o, sd, nil
}
