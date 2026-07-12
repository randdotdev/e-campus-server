package http

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/randdotdev/e-campus-server/internal/authz"
)

// VerifyMounts proves, after all routes are wired, that every route under
// protectedBase is gate-guarded and every standard action has a policy.
// It joins all violations into one error, so a failing boot reads as a
// checklist. An exempt entry is a path prefix; an entry preceded by a
// method ("POST /api/v1/applications") exempts exactly that route.
func (g *Gates) VerifyMounts(engine *gin.Engine, protectedBase string, exempt ...string) error {
	var errs []error
	for _, route := range engine.Routes() {
		if !strings.HasPrefix(route.Path, protectedBase) || isExempt(route.Method, route.Path, exempt) {
			continue
		}
		m, ok := g.mountFor(route.Path)
		if !ok {
			errs = append(errs, fmt.Errorf("authz: unguarded route %s %s", route.Method, route.Path))
			continue
		}
		errs = append(errs, verifyPolicyExists(m, route.Method, route.Path)...)
	}
	return errors.Join(errs...)
}

// mountFor finds the longest registered mount guarding the route. A mount
// guards a path only on a path-segment boundary: "/students" guards
// "/students" and "/students/:id", never "/students-archive".
func (g *Gates) mountFor(path string) (mount, bool) {
	var best mount
	found := false
	for _, m := range g.mounts {
		if !strings.HasPrefix(path, m.basePath) {
			continue
		}
		if rest := path[len(m.basePath):]; rest != "" && rest[0] != '/' {
			continue
		}
		if len(m.basePath) > len(best.basePath) {
			best, found = m, true
		}
	}
	return best, found
}

// verifyPolicyExists checks that the (resource, action) a route reaches has
// a compiled-in default policy, attributing the action the way the route's
// mount kind does. Custom colon actions are dispatched at runtime and cannot
// be enumerated from the route table; they are covered by the
// policy-invariant tests instead.
func verifyPolicyExists(m mount, method, path string) []error {
	var action authz.Action
	var ok bool
	switch m.kind {
	case mountSingleton:
		action, ok = singletonAction(method)
	case mountUnder:
		action, ok = underAction(method)
	default:
		hasTarget := hasParam(path, m.param)
		if method == "POST" && hasTarget {
			return nil // a custom colon method; not enumerable here
		}
		action, ok = verbAction(method, hasTarget)
	}
	if !ok {
		return nil
	}
	if _, exists := authz.DefaultPolicies()[authz.PolicyKey{Resource: m.resource, Action: action}]; !exists {
		return []error{fmt.Errorf("authz: route %s %s needs policy (%s, %s) but no default exists",
			method, path, m.resource, action)}
	}
	return nil
}

// hasParam reports whether the route pattern carries the param as a whole
// segment — ":id" must not match ":offeringId".
func hasParam(path, param string) bool {
	for _, seg := range strings.Split(path, "/") {
		if seg == ":"+param {
			return true
		}
	}
	return false
}

func isExempt(method, path string, exempt []string) bool {
	for _, e := range exempt {
		if m, p, ok := strings.Cut(e, " "); ok {
			if m == method && p == path {
				return true
			}
			continue
		}
		if strings.HasPrefix(path, e) {
			return true
		}
	}
	return false
}
