package platformauth

// InsecureDevSigningKey is the fallback JWT_SIGNING_KEY every service in
// this module uses when the JWT_SIGNING_KEY env var is unset. Previously
// hand-copied as an identical string literal into all 4 services'
// cmd/main.go, each with a comment warning it must stay byte-identical —
// exactly the kind of duplication this package exists to remove. It is
// obviously not a secret: it's committed, identical across every service,
// and logged loudly on every use (see each service's jwtSigningKey()
// helper in cmd/main.go). Exists purely so a zero-config single-command
// dev/preview run (e.g. this repo's .claude/launch.json, whose format has
// no way to inject an env var) has services able to talk to each other out
// of the box. Anything beyond solo local dev/preview must set a real
// JWT_SIGNING_KEY explicitly — see architecture.md Section 15.
const InsecureDevSigningKey = "INSECURE-DEV-ONLY-SHARED-SIGNING-KEY-DO-NOT-USE-BEYOND-LOCAL-PREVIEW"
