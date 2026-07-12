// Package http is the translator between HTTP and the authz engine — the
// "shape" and "act" steps of the request's life drawn in authz.go. The
// engine thinks; this package only converts.
//
// Shaping: a gate mounted on a route group turns each request into the
// engine's question. The resource is fixed at mount time; the action comes
// from the HTTP verb, or from the URL's colon suffix for custom methods
// (POST /students/:id:activate); the target is the ":id" path param.
// Anything malformed is answered 404 before evaluation starts.
//
// Acting: a deny becomes 403 (404 when the target does not exist); an allow
// stashes the Decision as the request's AccessInfo (access.go), writes the
// audit line, and lets the handler run.
//
// The package also carries the boot checklist proving every protected route
// is gated (checklist.go) and the policy administration endpoints
// (policy.go).
package http
