package http_test

import (
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/announcements"
	announcementshttp "github.com/randdotdev/e-campus-server/internal/announcements/http"
	announcementspg "github.com/randdotdev/e-campus-server/internal/announcements/postgres"
	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
)

// TestRoutesRegisterAnnouncementsEndpoints wires the handler exactly like
// cmd/api and freezes the announcements route table — the frontend's
// contract. Cross-context ports (mutes, notifier, files) stay nil: route
// registration never invokes them.
func TestRoutesRegisterAnnouncementsEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &sqlx.DB{}
	slogger := slog.Default()

	postSvc := announcements.NewPostService(
		announcementspg.NewPostRepository(db),
		announcementspg.NewUserLookup(db),
		announcementspg.NewScopeChecker(db),
		nil, nil, nil, slogger,
	)
	activitySvc := announcements.NewActivityService(
		announcementspg.NewActivityRepository(db),
		announcementspg.NewPublisherChecker(db),
		nil, nil, slogger,
	)
	gates := authzhttp.NewGates(authz.NewService(authz.StaticPolicyStore{}, nil, slogger), slogger)

	h := announcementshttp.NewHandler(postSvc, activitySvc, gates, zap.NewNop())

	router := gin.New()
	protected := router.Group("/api/v1")
	h.Routes(protected)

	got := make(map[[2]string]bool)
	for _, r := range router.Routes() {
		got[[2]string{r.Method, r.Path}] = true
	}

	want := [][2]string{
		{"DELETE", "/api/v1/activities/:id"},
		{"DELETE", "/api/v1/activity-attachments/:id"},
		{"DELETE", "/api/v1/post-attachments/:id"},
		{"DELETE", "/api/v1/posts/:id"},
		{"DELETE", "/api/v1/posts/:id/like"},
		{"GET", "/api/v1/activities"},
		{"GET", "/api/v1/activities/:id"},
		{"GET", "/api/v1/activities/:id/attachments/:attachmentId"},
		{"GET", "/api/v1/activities/:id/translation"},
		{"GET", "/api/v1/posts"},
		{"GET", "/api/v1/posts/:id"},
		{"GET", "/api/v1/posts/:id/attachments/:attachmentId"},
		{"GET", "/api/v1/posts/:id/comments"},
		{"POST", "/api/v1/activities"},
		{"POST", "/api/v1/activities/:id/attachments"},
		{"POST", "/api/v1/posts"},
		{"POST", "/api/v1/posts/:id"},
		{"POST", "/api/v1/posts/:id/comments"},
		{"POST", "/api/v1/posts/:id/like"},
		{"PUT", "/api/v1/activities/:id"},
		{"PUT", "/api/v1/activities/:id/pin"},
		{"PUT", "/api/v1/posts/:id"},
	}

	for _, w := range want {
		if !got[w] {
			t.Errorf("route missing: %s %s", w[0], w[1])
		}
	}
	if len(got) != len(want) {
		t.Errorf("route count = %d, want %d (unexpected route added or removed)", len(got), len(want))
	}
}
