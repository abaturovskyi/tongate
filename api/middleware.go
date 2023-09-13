package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/abaturovskyi/tongate/config"
	"github.com/uptrace/bunrouter"
)

func AuthMiddleware() bunrouter.MiddlewareFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			if !checkToken(req, config.Config.AdminToken) {
				w.WriteHeader(http.StatusUnauthorized)
				return nil
			}
			return next(w, req)
		}
	}
}

func HeadersMiddleware() bunrouter.MiddlewareFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			w.Header().Add("Content-Type", "application/json")

			return next(w, req)

		}
	}
}

func checkToken(req bunrouter.Request, token string) bool {
	auth := strings.Split(req.Header.Get("authorization"), " ")
	if len(auth) != 2 {
		return false
	}
	if auth[0] != "Bearer" {
		return false
	}
	if x := subtle.ConstantTimeCompare([]byte(auth[1]), []byte(token)); x == 1 {
		return true
	} // constant time comparison to prevent time attack
	return false
}
