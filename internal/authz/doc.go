// Package authz decides, before any handler runs, whether a person may do
// what they are asking — in one place, by one set of rules, failing
// closed. It evaluates three kinds of authority: staff (a position in the
// level × domain × scope space), offering seats (teacher, assistant,
// student, observer), and file relations (owner, writer, reader); the
// policy for an action says which of them may answer.
//
// Reading order: authz.go (the engine and its ports), policy.go (what a
// policy is and where policies live), staff.go, classroom.go, fs.go (the
// three checks), policy_defaults.go (the factory table), http/ (gates,
// stashed access info, the boot proof), postgres/ and redis/ (adapters).
//
// Changing who may do something is one reviewed line in
// policy_defaults.go; running deployments in DB mode tune rows via the
// admin API or reset to defaults. Adding a resource: declare the Entity,
// give it default entries, mount a gate — the boot checklist fails loudly
// on whatever is forgotten. The policy mode is one wiring line in main.go:
// StaticPolicyStore (code-only; edits refused) or the DB store behind the
// Redis cache (admin-tunable; seeded at boot).
//
// Never: gate the policy-admin endpoints with stored rows (recovery must
// be unlosable), hard-delete permission rows (boot resurrects the pair),
// or base a write decision on stashed AccessInfo facts (§14).
package authz
