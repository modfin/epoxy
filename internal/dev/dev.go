package dev

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/pkg/epoxy"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"strings"
	"time"
)

type contextKey struct{}

const cookieName = "epoxy-dev"

func Middleware(bcryptHash string, sessionDuration time.Duration, jwtEc256 *ecdsa.PrivateKey, jwtEc256Pub *ecdsa.PublicKey, devDisableSecure bool) epoxy.Middleware {
	if bcryptHash == "" {
		log.New().Fatal("dev: bcrypt hash required")
	}
	if sessionDuration.Milliseconds() <= 0 {
		log.New().Fatal("dev: session duration negative or zero")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			if err == nil && cookie.Valid() == nil {
				var c claims
				t, err := jwt.ParseWithClaims(cookie.Value, &c, func(t *jwt.Token) (interface{}, error) {
					if t.Method.Alg() != jwt.SigningMethodES256.Alg() {
						return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
					}
					return jwtEc256Pub, nil
				})
				if err == nil && t.Valid {
					log.New().WithField("dev_email", c.DevEmail).WithField("dev", "session active").AddToContext(r.Context())
					ctx := context.WithValue(r.Context(), contextKey{}, c.DevEmail)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			if r.Method == http.MethodPost {
				email := r.FormValue("email")
				password := r.FormValue("password")
				if bcrypt.CompareHashAndPassword([]byte(bcryptHash), []byte(password)) == nil {
					exp := time.Now().Add(sessionDuration)
					devClaims := claims{
						DevEmail: email,
						RegisteredClaims: jwt.RegisteredClaims{
							Issuer:    "epoxy-dev",
							IssuedAt:  &jwt.NumericDate{Time: time.Now()},
							ExpiresAt: &jwt.NumericDate{Time: exp},
						},
					}
					devJwt, err := jwt.NewWithClaims(jwt.SigningMethodES256, devClaims).SignedString(jwtEc256)
					if err != nil {
						log.New().WithError(err).AddToContext(r.Context())
						w.WriteHeader(http.StatusUnauthorized)
						return
					}

					c := &http.Cookie{
						Name:     cookieName,
						Value:    devJwt,
						Path:     "/",
						Expires:  exp,
						Secure:   !(r.URL.Host == "localhost" || devDisableSecure),
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					}
					http.SetCookie(w, c)
					log.New().WithField("dev_email", email).WithField("dev", "success, set cookie").AddToContext(r.Context())
					http.Redirect(w, r, r.URL.String(), http.StatusFound)
					return
				} else {
					log.New().WithField("dev_email", email).WithField("dev", "wrong password").AddToContext(r.Context())
					w.WriteHeader(http.StatusUnauthorized)
				}
			}
			log.New().WithField("dev", "not logged in, rendering form").AddToContext(r.Context())
			_, _ = io.Copy(w, strings.NewReader(page))
			return
		})
	}
}

type claims struct {
	DevEmail string `json:"dev_email"`
	jwt.RegisteredClaims
}

func Email(ctx context.Context) (string, error) {
	if u, ok := ctx.Value(contextKey{}).(string); ok {
		return u, nil
	}
	return "", errors.New("couldn't get dev email from context, make sure dev.Middleware has run")
}

const page = `
<!doctype html>
<html>
<head>
   <meta name="viewport" content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
   <title>Epoxy Dev Login</title>
</head>
<body>
	<form method="post" action="/">
		<div>
			<label for="email">Email</label>
			<input type="email" id="email" name="email">
		</div>
		<div>
			<label for="password">Password</label>
			<input type="password" id="password" name="password">
		</div>
		<input type="submit" />
	</form>
</body>
</html>
<style>
form {
	margin-top: 100px;
	margin-left: auto;
    margin-right: auto;
    width: 250px;
	display: flex;
    flex-direction: column;
	gap: 10px;
}
form div {
	display: flex;
	justify-content: space-between;
}
</style>
`
