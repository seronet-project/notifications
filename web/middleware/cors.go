package middleware

import (
    "net/http"

    "github.com/cloudfoundry-incubator/notifications/config"
    "github.com/ryanmoran/stack"
)

type CORS struct{}

func NewCORS() CORS {
    return CORS{}
}

func (ware CORS) ServeHTTP(w http.ResponseWriter, req *http.Request, context stack.Context) bool {
    env := config.NewEnvironment()
    w.Header().Set("Access-Control-Allow-Origin", env.CORSOrigin)
    w.Header().Set("Access-Control-Allow-Methods", "GET, PATCH")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")

    return true
}