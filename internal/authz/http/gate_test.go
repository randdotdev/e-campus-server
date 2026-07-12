package http

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
)

// attributeOn runs attribute against a real gin route, returning what a gate
// would see for the request.
func attributeOn(t *testing.T, method, pattern, path string) (*AccessInfo, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var info *AccessInfo
	var ok bool
	router.Handle(method, pattern, func(c *gin.Context) {
		info, ok = attribute(c, authz.ResourceStudent)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(method, path, nil))
	return info, ok
}

func TestAttribute(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name       string
		method     string
		pattern    string
		path       string
		wantOK     bool
		wantAction authz.Action
		wantTarget bool
	}{
		{"get_one", "GET", "/students/:id", "/students/" + id.String(), true, authz.ActionGet, true},
		{"list", "GET", "/students", "/students", true, authz.ActionList, false},
		{"create", "POST", "/students", "/students", true, authz.ActionCreate, false},
		{"update", "PUT", "/students/:id", "/students/" + id.String(), true, authz.ActionUpdate, true},
		{"delete", "DELETE", "/students/:id", "/students/" + id.String(), true, authz.ActionDelete, true},
		{"custom_action", "POST", "/students/:id", "/students/" + id.String() + ":activate", true, authz.Action("activate"), true},
		{"custom_action_needs_post", "PUT", "/students/:id", "/students/" + id.String() + ":activate", false, "", false},
		{"bare_post_to_dispatcher", "POST", "/students/:id", "/students/" + id.String(), false, "", false},
		{"trailing_colon", "POST", "/students/:id", "/students/" + id.String() + ":", false, "", false},
		{"action_without_id", "POST", "/students/:id", "/students/:activate", false, "", false},
		{"non_uuid_id", "GET", "/students/:id", "/students/nope", false, "", false},
		{"multi_colon_unknown_action", "POST", "/students/:id", "/students/" + id.String() + ":a:b", true, authz.Action("a:b"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := attributeOn(t, tt.method, tt.pattern, tt.path)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if info.action != tt.wantAction {
				t.Fatalf("action = %q, want %q", info.action, tt.wantAction)
			}
			if (info.targetID != uuid.Nil) != tt.wantTarget {
				t.Fatalf("target presence = %v, want %v", info.targetID != uuid.Nil, tt.wantTarget)
			}
		})
	}
}

// Note on multi_colon_unknown_action: attribution accepts "a:b" as an action
// string; the policy lookup then denies it (no policy names such an action),
// so the request 403s. Malformed never widens access.

func TestVerifyMountsFlagsUnguardedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gates := NewGates(nil, nil)

	guarded := router.Group("/api/v1/students")
	gates.register(guarded, authz.ResourceStudent, mountTarget, "id")
	guarded.GET("", func(*gin.Context) {})

	router.GET("/api/v1/rogue", func(*gin.Context) {})     // forgot the gate
	router.GET("/api/v1/me/things", func(*gin.Context) {}) // exempt by design

	err := gates.VerifyMounts(router, "/api/v1", "/api/v1/me")
	if err == nil {
		t.Fatal("want the rogue route reported")
	}
	if !strings.Contains(err.Error(), "rogue") {
		t.Fatalf("error must name the unguarded route, got: %s", err)
	}
	if strings.Contains(err.Error(), "/me/") {
		t.Fatal("exempt routes must not be reported")
	}
}

func TestMountForRespectsSegmentBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	gates := NewGates(nil, nil)
	gates.register(router.Group("/api/v1/students"), authz.ResourceStudent, mountTarget, "id")

	if _, ok := gates.mountFor("/api/v1/students/:id"); !ok {
		t.Fatal("child path must be guarded")
	}
	if _, ok := gates.mountFor("/api/v1/students"); !ok {
		t.Fatal("exact path must be guarded")
	}
	if _, ok := gates.mountFor("/api/v1/students-archive"); ok {
		t.Fatal("a lookalike prefix must NOT count as guarded")
	}
}
