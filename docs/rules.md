# Engineering Rules & AI Assistant Boundaries
## AI-Powered Cricket Decision Review System (Cricket DRS)

**Status:** Draft v1.1 — updated post CTO review
**Applies to:** all human engineers and any AI coding/agentic assistants working on this codebase

**Changelog (v1.0 → v1.1):** moves tenant-isolation and security testing to start in Phase 1 rather than
Phase 9, adds a data strategy / label-QA requirement for the retraining loop, reclassifies weather/lighting
variation as core training distribution rather than an edge case, adds anti-tampering threat-model
requirements, and clarifies that testing rigor ramps with project maturity rather than applying peak
enterprise rigor uniformly from day one.

---

## 0. A Note on Rigor vs. Phase (new — CTO review)

The standards in this document (80% coverage, full E2E suites, security audits, load tests) are the
**target state for anything reaching production**, but are not all expected to be fully in force during
Phase 0–2 feasibility and infrastructure work, where the priority is validating core technical
assumptions quickly. Specifically:
- **Security fundamentals (tenant isolation testing, secrets management, no hardcoded credentials)
  apply from Phase 1, no exceptions** — these are cheap to do right from the start and expensive to
  retrofit (see Section 6.5).
- **Full coverage/load/security-audit rigor** ramps up through Phase 3–9 as components move from
  research/prototype to production-facing, per `phases.md`.
- This section exists so "move fast in Phase 0" and "80% coverage is mandatory" don't read as
  contradictory — they apply to different kinds of code at different times, and the distinction should
  be explicit rather than each engineer guessing.

---

## 1. Development Principles

- **Clean Architecture / Hexagonal boundaries:** domain logic (decision rules, trajectory math, review workflow state machine) must be isolated from framework, database, and transport concerns. No domain class imports a web framework or ORM type directly.
- **SOLID principles** are mandatory, not aspirational — enforced in code review, not just documentation:
  - Single Responsibility: a service/class does one thing (e.g., the LBW engine does not also handle notification dispatch).
  - Open/Closed: new decision types (e.g., a future "boundary catch" reviewer) should be addable without modifying the core review orchestration state machine.
  - Liskov, Interface Segregation, Dependency Inversion: enforced via interface-first design in the core application services.
- **Maintainable and testable by construction:** if a component can't be unit-tested without spinning up a database or GPU, its boundaries are wrong — restructure before proceeding, don't work around it with brittle integration tests only.
- **Production-quality from day one for anything touching a live decision path.** Prototype-quality code is acceptable only in clearly labeled experimentation/notebook environments (`/research`), never in services that feed the Review Orchestration Service.

---

## 2. Coding Rules

### 2.1 Naming Conventions
- Go services: standard Go conventions (`PascalCase` for exported identifiers, `camelCase` for unexported, short/idiomatic package names — no `utils`/`common` dumping-ground packages). Packages reflect bounded context (`review`, `match`, `identity`), not technical layer (a top-level `controllers` or `handlers` package spanning multiple domains is disallowed). Errors are handled explicitly per Go idiom (returned, not panicked, except for genuinely unrecoverable startup failures) — this ties directly into Section 4.2 below.
- Python (CV/ML): PEP8, `snake_case` for functions/variables, `PascalCase` for classes. Model artifacts named `{model_type}_{version}_{tier}.pt` (e.g., `ball_detector_v3_broadcast.pt`).
- React/TypeScript: `PascalCase` components, `camelCase` hooks/functions prefixed appropriately (`useReviewState`), one component per file matching filename.
- Database: `snake_case` table/column names, singular table names avoided in favor of plural (`matches`, `review_decisions`) for consistency.
- Every domain concept (e.g., "review," "decision package," "confidence score") must use the **same term** across code, API contracts, and documentation — no synonyms drifting across layers (a direct instance of avoiding unnecessary complexity).

### 2.2 Folder Structure
- Monorepo recommended for the modular monolith + CV pipeline during early phases (simplifies cross-cutting refactors and shared type contracts); split to polyrepo only when team/deployment independence justifies it.
- Top-level structure organized by bounded context, not technical layer:
```
/services            (Go)
  /review-orchestration
  /match-tournament
  /identity-access
  /analytics-reporting
/edge-agent          (Go — venue capture/buffer/sync agent)
/ml-pipeline         (Python)
  /ball-tracking
  /edge-detection
  /lbw-engine
  /runout-detection
  /model-registry
/apps
  /web-app           (TypeScript/React)
  /umpire-console     (TypeScript/React)
  /mobile-app         (Dart/Flutter)
/infra
  /terraform
  /k8s
/docs
```
- Each service directory is self-contained: its own tests, README, and (where applicable) OpenAPI/proto contract file.

### 2.3 Documentation Requirements
- Every service has a `README.md`: purpose, how to run locally, key dependencies, owning team.
- Every non-trivial architectural decision gets an **ADR (Architecture Decision Record)** in `/docs/adr/`, including context, options considered, decision, and consequences. This is mandatory for anything affecting the AI/ML pipeline's accuracy or latency characteristics.
- Public and partner-facing APIs must be documented via OpenAPI (REST) or `.proto` files (gRPC) kept in sync with implementation — CI fails the build if contract and implementation diverge (contract testing).

### 2.4 Code Review Standards
- No direct pushes to `main`/`release` branches; all changes via PR with at least one approving review.
- PRs touching the CV/ML decision pipeline, LBW engine, or review orchestration state machine require review from someone with domain context in that area — not just "any available reviewer."
- PR description must state: what changed, why, how it was tested, and any accuracy/latency impact if applicable.
- Reviewers explicitly check for: scope creep, missing tests, unexplained complexity, and any change to the AI-decision-boundary rules in Section 5 below.

---

## 3. Technology Rules

### 3.1 Approved Programming Languages
- **Go** — core application/orchestration services and the venue edge capture/buffer agent.
- **Python** — CV/ML pipeline, model training, data science/experimentation.
- **TypeScript** — all frontend (web, umpire console) and Node-based tooling.
- **Dart** — mobile app (Flutter).
- **HCL (Terraform)** — infrastructure as code.
- Any language outside this list requires an ADR justifying the exception before use in a shared/production codebase.

### 3.2 Approved Frameworks
- Backend: standard library `net/http` + a minimal router (e.g., `chi`) preferred over heavy full-stack Go frameworks, in keeping with Go idiom and the "avoid unnecessary complexity" principle; gRPC (`google.golang.org/grpc`) for internal service-to-service contracts; `sqlc` or `pgx` (not a heavy ORM) for PostgreSQL access, since explicit queries suit the review/audit data path better than ORM-generated ones (Section 3.4). FastAPI (Python) for internal ML service APIs.
- Frontend: React + TypeScript (web), Flutter (mobile).
- ML: PyTorch (primary), OpenCV (CV primitives). TensorFlow is not disallowed but PyTorch is the default to avoid maintaining two ML ecosystems without justification.
- Testing: Go's built-in `testing` package + `testify` for assertions/mocks (Go), Pytest (Python), Jest/React Testing Library (frontend), Playwright (E2E).

### 3.3 Approved Libraries — General Rule
Any new third-party dependency must satisfy:
1. Actively maintained (commit activity within the last 12 months).
2. Compatible license (MIT/Apache2/BSD preferred; GPL/AGPL requires legal review before use in shipped code).
3. No known unpatched critical CVEs at time of adoption.
4. Justified need — not "it looked convenient." Prefer the standard library or an already-adopted dependency first.

### 3.4 Libraries/Practices to Avoid, and Why
- **Avoid ad-hoc/home-grown cryptography or auth** — always use vetted libraries (Spring Security, well-established OAuth2/OIDC providers). Rationale: security-critical code is exactly where "not invented here" causes real harm.
- **Avoid unpinned dependency versions** in any production service — reproducibility and supply-chain security require lockfiles/pinned versions everywhere (`pom.xml`, `requirements.txt`/`poetry.lock`, `package-lock.json`).
- **Avoid heavy ORMs with implicit lazy-loading/hidden-query magic**, full stop — this is largely a non-issue given the Go default of explicit SQL via `sqlc`/`pgx` (Section 3.2), but applies doubly to the review/audit data path, where decision-record immutability and query predictability matter more than any convenience a heavier abstraction might offer.
- **Avoid introducing a new database technology** without an ADR showing the existing stack (PostgreSQL/TimescaleDB) is genuinely insufficient — directly enforces "never create unnecessary complexity."
- **Avoid framework-of-the-month adoption** for the frontend or backend — stability and long-term maintainability outweigh marginal DX improvements from chasing new frameworks.

---

## 4. Error Handling

### 4.1 Logging Standards
- Structured (JSON) logging everywhere — no unstructured `print`/string-concatenated logs in production code.
- Every log line in the review pipeline includes: `match_id`, `review_id` (once assigned), `service_name`, `trace_id`.
- Log levels used consistently: `ERROR` (requires attention/paging), `WARN` (degraded but functioning, e.g., low-confidence flagged), `INFO` (key lifecycle events), `DEBUG` (dev/troubleshooting only, disabled by default in production).
- **Never log raw video/biometric data** or full PII in application logs — log references/IDs, not the sensitive payload itself.

### 4.2 Error Handling (Go idiom)
- Errors are explicit return values, per Go convention — never discarded with `_`. Domain-specific sentinel/wrapped error types (e.g., `ErrLowConfidenceTracking`, `ErrCameraSyncFailure`) rather than generic errors bubbling to the API boundary, so callers can distinguish "AI couldn't produce a confident answer" from "system failure" via `errors.Is`/`errors.As`.
- `panic`/`recover` is reserved for genuinely unrecoverable conditions (e.g., failed startup configuration) — never used as a substitute for normal error-flow control, and never allowed to cross a service boundary unrecovered in a request-handling path.
- No silent error discarding. Every returned error is either handled meaningfully, wrapped with context (`fmt.Errorf("...: %w", err)`) and propagated, or logged at an appropriate level — never dropped.
- The Review Orchestration Service must explicitly handle the "AI pipeline failed or timed out" case as a first-class flow: fall back to "manual review, no AI assist," never leave the umpire console in an ambiguous/hung state.

### 4.3 Monitoring
- Golden signals (latency, traffic, errors, saturation) instrumented for every service via a standard observability stack (e.g., OpenTelemetry → Prometheus/Grafana, or a managed equivalent).
- Dedicated dashboards for the review-pipeline latency budget (per tier, see `architecture.md` NFRs) with alerting on SLA-approaching breaches, not just outright failures.
- Model-specific monitoring: prediction confidence distribution, override rate (human disagrees with AI), and detection-failure rate tracked over time as leading indicators of model drift.

### 4.4 Failure Recovery Strategy
- Graceful degradation is a hard requirement for the live-match path: if the AI pipeline is unavailable, the system must clearly signal "AI assist unavailable, proceed with manual review" rather than blocking the match or presenting stale/wrong data.
- Idempotent review-processing jobs (safe to retry) given the message-bus-driven architecture; deduplication keyed on `review_id`.
- Circuit breakers between the Review Orchestration Service and the ML pipeline, so a struggling/overloaded ML service degrades that capability without cascading failure into match administration or analytics services.
- Documented runbooks for the top failure modes (camera desync, ML service overload, ingest gateway outage) — required before broadcast-tier go-live for any venue.

---

## 5. AI Boundaries

This section governs both (a) the AI/ML system's role within the product, and (b) any AI coding assistant contributing to this codebase.

### 5.1 Product AI Boundaries (Ball-tracking/decision models)

- **AI provides recommendations with evidence; humans make the final call, always, in live matches.** The Review Orchestration Service architecture must make it structurally impossible to auto-finalize a decision without a human confirmation step during live play (see `prd.md` Section 4.3, `architecture.md` Section 3).
- **AI may make autonomous decisions only in clearly non-authoritative contexts**: e.g., auto-generating a highlight clip, auto-flagging a low-confidence detection for review, auto-populating a draft analytics report. Anything that changes the outcome of a match must have a human decision point.
- **Confidence must always be surfaced, never hidden.** A "we're not sure" result routed to mandatory manual review is a correct system behavior, not a failure state to be engineered away.
- **No silent model updates during a live tournament.** Model version changes are deployed and validated between matches/seasons, with a documented rollback plan, never hot-swapped mid-match.
- **Avoid hallucinated assumptions:** the LBW/edge/run-out models must never fabricate a confident answer from insufficient/occluded data — the QA/confidence layer's job is specifically to catch and flag this rather than let a model "guess smoothly."

### 5.2 AI Coding Assistant Boundaries (for this repository)

- An AI assistant (e.g., Claude Code or similar) may write, refactor, and test code, but:
  - Must always explain the architectural reasoning for non-trivial changes (per `userPreferences`) rather than silently applying a pattern.
  - Must not introduce a new dependency, database technology, or cross-cutting architectural pattern without flagging it explicitly for human approval (aligns with Section 3.4).
  - Must not modify the review-decision immutability guarantees, RBAC/auth logic, or the human-confirmation gate in the review workflow without an explicit human-authored ADR justifying the change — these are safety-critical boundaries, not normal refactor targets.
  - Must flag, not silently resolve, any ambiguity in requirements that affects decision-accuracy behavior (e.g., "what should happen if two cameras disagree by more than X mm") — this is a product/domain decision, not an implementation detail.
- Human engineers remain accountable for all merged code; AI-authored PRs go through the same review bar as human-authored ones, with no exception for "the AI already tested it."

---

## 6. Testing Standards

### 6.1 Unit Testing
- Minimum 80% line coverage on core application services and the review orchestration state machine; coverage is a signal, not a target to game — meaningful assertions over trivial getter/setter tests.
- Domain logic (decision rules, quota enforcement, RBAC policy evaluation) must be unit-testable in isolation, without a running database or network call (enforced by the Clean Architecture boundary in Section 1).

### 6.2 Integration Testing
- Contract tests between services (especially Review Orchestration ↔ ML pipeline, and all public/partner APIs) run in CI against the published OpenAPI/proto contracts.
- End-to-end tests (Playwright) cover the critical umpire review journey (trigger → evidence display → confirm/override → persisted decision) against a staging environment with representative synthetic footage.

### 6.3 AI Model Validation
- Every model version is validated against a held-out, human-labeled benchmark dataset before promotion, with accuracy/precision/recall and calibration metrics recorded in the model registry (see `architecture.md` Section 8).
- **No model reaches production without a documented accuracy report per tier** (broadcast vs accessible), since their expected accuracy ceilings genuinely differ.
- Regression testing: new model versions are evaluated against the same historical benchmark set as prior versions to catch silent accuracy regressions before rollout.
- **Lighting/weather variation is core training distribution, not an adversarial edge case (revised — CTO review).** For accessible-tier club cricket, evening/weekend matches under inconsistent floodlighting are the realistic median case, not a rare occurrence. The benchmark and training sets must include this variation from the outset; a separate, additionally-maintained adversarial set still covers genuinely rare cases (motion blur extremes, pink ball, non-standard kit colors).
- **Class imbalance / rare-event handling (new — CTO review):** edges, run-outs, and close LBWs are simultaneously the highest-value and rarest events in any dataset. Benchmark and training data collection must deliberately over-sample or target these cases (e.g., dedicated data-collection sessions focused on close dismissals), rather than relying on naturally-occurring frequency, which would bias models toward the easy, obvious cases.
- **Human-override label QA (new — CTO review):** the Phase 8 retraining loop uses umpire overrides as candidate new training labels (`architecture.md` Section 8). An override reflects a human decision under pressure, not verified ground truth. Override events must pass an **adjudication step** (e.g., expert panel review, or cross-referencing against the highest-confidence available evidence) before being promoted into the training set — never piped directly from "umpire overrode" to "used to retrain the model."

### 6.4 Performance Testing
- Load testing of the review pipeline under concurrent-match conditions before any broadcast-tier commitment, validating the latency SLA (`architecture.md` Section 11) holds under realistic peak load (e.g., simultaneous reviews across N concurrent matches).
- Soak testing for memory/resource leaks in long-running services (a multi-day tournament is a realistic sustained-load scenario, not just a burst test).

### 6.5 Security Testing
- Dependency/CVE scanning integrated into CI (fails build on critical unpatched vulnerabilities per Section 3.3).
- Static application security testing (SAST) on every PR; dynamic testing (DAST) against staging before major releases.
- Periodic penetration testing (at minimum annually, and before any broadcast/board-tier commercial launch) given the governance/audit-trail sensitivity of this system.
- **Tenant-isolation testing starts in Phase 1 (revised — CTO review), not Phase 9.** It explicitly verifies no cross-tenant data leakage in analytics and admin endpoints (directly tied to the multi-tenant architecture in `architecture.md` Section 15), and runs as part of the standard CI suite from the Identity & Access Service's first implementation, not as a pre-launch checklist item.
- **Secrets management verification (new — CTO review):** CI includes a check that fails the build if credentials appear in code, config files, or container image layers, enforcing `architecture.md` Section 15's secrets-manager requirement.
- **Camera-feed integrity / anti-tampering testing (new — CTO review):** dedicated test scenarios simulating feed substitution, replay injection, and lens obstruction against the Media Ingest Gateway, validating the detection/alerting behavior specified in `architecture.md` Section 15 and `prd.md` Section 13.4. Owned by the team implementing the Media Ingest Gateway (Phase 2) and re-run whenever that service changes.
- **Umpire Console session-security testing (new — CTO review):** verifies short-lived sessions, mandatory re-authentication on confirm/override actions, and device-allowlisting behavior specified in `architecture.md` Section 15, exercised as part of the Phase 7 E2E suite.
