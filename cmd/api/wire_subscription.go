package main

import (
	"github.com/randdotdev/e-campus-server/internal/subscription"
	subscriptionhttp "github.com/randdotdev/e-campus-server/internal/subscription/http"
	subscriptionpg "github.com/randdotdev/e-campus-server/internal/subscription/postgres"
)

// subscriptionSet is what the subscription context exports: service feeds
// the limit checks of management and files.
type subscriptionSet struct {
	handler *subscriptionhttp.Handler
	service *subscription.Service
}

// wireSubscription builds the subscription context.
func wireSubscription(infra *infra) subscriptionSet {
	repo := subscriptionpg.NewRepository(infra.db)
	service := subscription.NewService(repo)
	return subscriptionSet{
		handler: subscriptionhttp.NewHandler(service, infra.log),
		service: service,
	}
}
