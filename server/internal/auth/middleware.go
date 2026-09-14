package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
)

type CredentialKind string

const (
	CredentialCookie CredentialKind = "cookie"
	CredentialBearer CredentialKind = "bearer"
)

type Credential struct {
	Token string
	Kind  CredentialKind
}

var ErrSessionCredential = errors.New("invalid or missing session credential")

type credentialContextKey struct{}

func SessionCredential(r *http.Request) (Credential, error) {
	if credential, ok := r.Context().Value(credentialContextKey{}).(Credential); ok {
		return credential, nil
	}

	authorizations := r.Header.Values("Authorization")
	cookie, cookieErr := r.Cookie(SessionCookieName)
	hasCookie := cookieErr == nil
	if len(authorizations) > 0 && hasCookie {
		return Credential{}, ErrSessionCredential
	}
	if len(authorizations) > 0 {
		if len(authorizations) != 1 {
			return Credential{}, ErrSessionCredential
		}
		parts := strings.Fields(authorizations[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			return Credential{}, ErrSessionCredential
		}
		return Credential{Token: parts[1], Kind: CredentialBearer}, nil
	}
	if hasCookie && cookie.Value != "" {
		return Credential{Token: cookie.Value, Kind: CredentialCookie}, nil
	}
	return Credential{}, ErrSessionCredential
}

func SessionToken(r *http.Request) string {
	credential, err := SessionCredential(r)
	if err != nil {
		return ""
	}
	return credential.Token
}

func WithSessionCredential(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		credential, err := SessionCredential(r)
		if err == nil {
			ctx := context.WithValue(r.Context(), credentialContextKey{}, credential)
			if credential.Kind == CredentialBearer {
				ctx = httpapi.WithBearerCredential(ctx)
			}
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
