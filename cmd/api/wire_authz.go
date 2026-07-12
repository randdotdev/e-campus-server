package main

import (
	"context"
	"fmt"
	"time"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	authzpg "github.com/randdotdev/e-campus-server/internal/authz/postgres"
	authzredis "github.com/randdotdev/e-campus-server/internal/authz/redis"
)

// authzSet is what the authz context exports. handler is nil in static
// policy mode — there is nothing to edit over HTTP.
type authzSet struct {
	service *authz.Service
	gates   *authzhttp.Gates
	handler *authzhttp.Handler
}

// wireAuthz builds the decision engine and gates. AUTHZ_POLICY_MODE picks
// where policies live; each branch wires everything its mode needs.
func wireAuthz(ctx context.Context, infra *infra) (authzSet, error) {
	readers := authzpg.NewReaders(infra.db)

	// Static mode: policies compiled into the binary; a change is a deploy.
	if !infra.cfg.Authz.PoliciesInDB() {
		service := authz.NewService(authz.StaticPolicyStore{}, readers, infra.slog)
		return authzSet{
			service: service,
			gates:   authzhttp.NewGates(service, infra.slog),
		}, nil
	}

	// DB mode: postgres policies behind the redis cache, seeded once from
	// the same defaults, editable at runtime through the authz handler.
	policyCache := authzredis.NewPolicyCache(authzpg.NewPolicyStore(infra.db), infra.rdb, infra.slog)
	service := authz.NewService(policyCache, readers, infra.slog)

	seedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	err := service.SeedPolicies(seedCtx)
	cancel()
	if err != nil {
		return authzSet{}, fmt.Errorf("seed authz policies: %w", err)
	}
	infra.log.Info("authz policies seeded (db mode)")

	return authzSet{
		service: service,
		gates:   authzhttp.NewGates(service, infra.slog),
		handler: authzhttp.NewHandler(service),
	}, nil
}
