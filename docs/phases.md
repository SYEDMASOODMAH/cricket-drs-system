# Implementation Roadmap
## AI-Powered Cricket Decision Review System (Cricket DRS)

**Status:** Draft v1.1 — updated post CTO review
**Companion to:** `prd.md`, `architecture.md`, `rules.md`

**Sequencing philosophy:** de-risk the hardest, most novel technical uncertainty (ball tracking + trajectory accuracy) as early as possible, before investing in breadth of features. Accessible tier (1–2 camera) is the MVP target; broadcast tier is a later scaling phase built on the same core, per `architecture.md` Section 1.

**Changelog (v1.0 → v1.1):** Phase 0 scope expanded to include a directional cost model, a real venue
connectivity assessment, and early business-model/willingness-to-pay validation — previously these were
implicitly deferred to Phase 10, which risked eight-plus phases of investment before testing core
commercial and infrastructure assumptions. Explicit go/no-go phase gates added before Phase 3, 5, and 9.
Consent capture and legal/privacy groundwork added as Phase 1 deliverables. See the CTO review for full
rationale.

---

## Phase 0 — Research & Product Definition

**Objective:** Validate the core technical hypothesis (can ball tracking + trajectory prediction reach usable accuracy from 1–2 consumer cameras) before committing to full build-out, finalize product scope, and validate the commercial/infrastructure assumptions the entire roadmap depends on.

- **Features delivered:** none (research spike only).
- **Technical tasks:**
  - Literature review + feasibility prototyping of monocular/dual-camera ball tracking and trajectory reconstruction.
  - Collect an initial labeled dataset (own-club footage from Montreal Overseas CC and MCA matches, with permission) for early model experimentation — with deliberate coverage of realistic lighting conditions (evening/weekend club matches), since these are core distribution, not edge cases (see `rules.md` Section 6.3).
  - Benchmark candidate object-detection architectures on cricket-ball-scale, high-speed objects.
  - Validate the physics-based LBW trajectory model against manually verified real deliveries.
  - **(New — CTO review) Real venue connectivity assessment:** measure actual uplink bandwidth at 3–5 representative MCA-tier venues across different times/days, to establish the local-inference-fallback threshold in `architecture.md` Section 9a — this is an empirical input the architecture explicitly depends on, not an assumption.
  - **(New — hardware model update) GoPro kit field validation:** battery/thermal behavior under continuous multi-hour recording (a full match is 3–4 hours), USB-C tethered webcam-mode capture reliability, and fisheye/lens-distortion calibration feasibility, tested on at least one real MCA-tier match — this informs whether USB-C tethered power solves the battery/heat risk in practice, not just on paper.
  - **(New — CTO review) Directional cost estimation:** rough per-match cost estimate (GPU inference + edge hardware + storage) per `architecture.md` Section 19, sized against realistic accessible-tier match volume. **(Revised — hardware model update)** now explicitly includes GoPro kit unit cost (~$600–2,000 per 2–4 camera kit) amortized across matches/seasons, since the platform owns and provides the hardware under the subscription model rather than the club bringing its own.
  - **(New — CTO review, revised — hardware model update) Business-model validation conversations:** direct conversations with MCA and 2–3 comparable leagues on willingness to pay for a **subscription that includes platform-owned hardware** (not a software-only price point), preferred pricing model (per-match/per-season), and practical questions like who's responsible for kit setup/teardown on match day.
- **Required skills:** ML/CV research engineer, sports-domain SME (product owner's own cricket expertise applies directly here), data labeling support.
- **Dependencies:** access to real match footage and a small labeled dataset; no production infrastructure dependency.
- **Deliverables:** feasibility report with disclosed accuracy ceiling estimates per tier, venue connectivity assessment, directional cost model, and a business-model validation summary.
- **Testing requirements:** benchmark methodology must be documented and repeatable; accuracy claims backed by a held-out test set, not training-set performance.
- **Completion criteria:** documented feasibility report reviewed and approved; initial dataset of ≥ 500 labeled ball-tracking clips assembled (explicitly a feasibility-validation sample, **not** a production-training-ready dataset — see Phase 3 data-scaling task).

### Phase 0 Exit Gate (new — CTO review)

Phase 1 does not begin until all of the following are true:
1. Feasibility report shows a plausible path to a usable (even if not yet final) accessible-tier accuracy ceiling — if the data suggests monocular tracking cannot plausibly clear a usable bar, the roadmap pivots (e.g., mandate 2-camera minimum) before Phase 1 infrastructure investment.
2. At least one MCA-tier venue's connectivity has been measured, informing the Section 9a fallback threshold.
3. At least directional cost-per-match figures exist and have been sanity-checked against Phase 0's business-model conversations — if the gap between plausible cost and plausible willingness-to-pay looks unbridgeable, that's a Phase 0 finding to act on, not a Phase 10 surprise.

---

## Phase 1 — Foundation & Architecture

**Objective:** Stand up the core platform skeleton, environments, and engineering practices so subsequent phases build on solid infrastructure.

- **Features delivered:** empty-shell platform: auth, multi-tenant org/user model, CI/CD, environments; legal/consent groundwork.
- **Technical tasks:**
  - Repo/monorepo setup per `rules.md` Section 2.2 folder structure.
  - Identity & Access Service (multi-tenant RBAC) implementation.
  - Match & Tournament Service skeleton (data model only, no video features yet).
  - CI/CD pipelines (build, test, SAST/dependency scanning per `rules.md` Section 6.5).
  - Infrastructure-as-code baseline (Terraform modules for core cloud resources, Kubernetes cluster bootstrap).
  - Observability stack setup (structured logging, metrics, tracing baseline).
  - **(New — CTO review) Secrets manager stood up** (`architecture.md` Section 15) before any real credential is created — no environment-variable/committed-config credentials from day one.
  - **(New — CTO review) Tenant-isolation test suite established** alongside the Identity & Access Service's first implementation, not deferred to Phase 9 (`rules.md` Section 6.5).
  - **(New — CTO review) Consent capture data model and flow** (`prd.md` Section 5.6, 13.3) built into the Match & Tournament / Identity & Access data model from the start, so it isn't retrofitted once real player data exists.
  - **(New — CTO review) Legal review kickoff:** liability/ToS language and Law 25/PIPEDA compliance review (`prd.md` Section 13.1–13.2) initiated in parallel with engineering — this has real lead time and should not start in Phase 9.
- **Required skills:** platform/DevOps engineer, backend engineer (Go), security engineer (part-time review), legal/privacy counsel (part-time/consulting, new — CTO review).
- **Dependencies:** Phase 0 go decision; cloud provider account provisioned.
- **Deliverables:** deployable empty platform across dev/staging environments; ADR log started; CI green on a trivial service.
- **Testing requirements:** unit test scaffolding in place; contract-testing framework validated with a trivial example service.
- **Completion criteria:** a new engineer can clone the repo, run the platform locally/in dev, and deploy a change through CI/CD to staging within their first week.

---

## Phase 2 — Video Ingestion & Camera Management

**Objective:** Build the venue-to-cloud video pipeline and camera calibration capability for both accessible and broadcast tiers.

- **Features delivered:** camera registration, edge capture/buffer agent, ingest gateway, basic calibration workflow, GoPro kit provisioning workflow.
- **Technical tasks:**
  - **(Revised — hardware model update)** Edge capture/buffer agent implements **USB-C tethered capture** (UVC webcam-mode read via `v4l2`/OpenCV, with HDMI+capture-card as a documented fallback for camera models without webcam mode) for venue-local camera ingestion, plus SRT/WebRTC for the separate edge-box-to-cloud upload leg, per `architecture.md` Section 10.
  - Media Ingest Gateway service; object storage integration with lifecycle policies.
  - Multi-camera time synchronization approach (software/audio-fingerprint sync for accessible tier).
  - Camera Calibration Service: standardized GoPro-class lens-distortion profile (built once) plus per-venue extrinsic calibration and per-camera-unit registration/ID (`prd.md` Section 13.5), replacing the earlier "calibrate arbitrary phone lenses" approach.
- **Required skills:** CV engineer (calibration/sync expertise), backend engineer, embedded/edge software engineer.
- **Dependencies:** Phase 1 platform; Phase 0 GoPro kit field validation findings; access to real venue(s) for field testing (Montreal Overseas CC home ground is a natural first test venue).
- **Deliverables:** working end-to-end capture → buffer → cloud-ingest pipeline validated at a real club venue, using the actual platform-provided GoPro kit (not a phone stand-in).
- **Testing requirements:** field test across at least 2 differing venue/lighting/connectivity conditions; calibration accuracy validated against measured pitch geometry; anti-tampering test scenarios per `rules.md` Section 6.5 exercised against the Media Ingest Gateway; USB-C tethered capture reliability re-validated at production scale (Phase 0 was a single-match spot check).
- **Completion criteria:** a real match can be recorded and a clip successfully retrieved on-demand from the buffer with correct multi-camera synchronization within a defined tolerance, using the GoPro kit end-to-end.
- **(New — CTO review)** Local-inference fallback path (`architecture.md` Section 9a) implemented and validated using the bandwidth threshold established in Phase 0's connectivity assessment.

---

## Phase 3 — Ball Tracking Engine

**Objective:** Productionize the ball detection and trajectory tracking pipeline validated in Phase 0 research.

- **Features delivered:** Ball Detection & Tracking Service (production-grade, not research prototype).
- **Technical tasks:**
  - Production object-detection model training/tuning pipeline; model registry integration.
  - Multi-view triangulation (broadcast) and monocular-assisted estimation (accessible) trajectory reconstruction.
  - Confidence & QA Layer v1 (plausibility checks, low-confidence flagging).
  - Performance optimization to meet per-tier latency budgets (`architecture.md` Section 11).
  - **(New — CTO review) Data scaling plan:** expand from Phase 0's ~500-clip feasibility sample to a production-scale training set, with deliberate class-balancing for rare events (edges, run-outs, close LBWs) and lighting/weather variation treated as core distribution per `rules.md` Section 6.3.
- **Required skills:** ML engineer, CV engineer, MLOps/infrastructure engineer.
- **Dependencies:** Phase 2 ingestion pipeline delivering usable synchronized footage; expanded labeled dataset.
- **Deliverables:** Ball Detection & Tracking Service deployed to staging, meeting accuracy targets from `prd.md` FR2 on a held-out validation set.
- **Testing requirements:** accuracy benchmark report per tier per `rules.md` Section 6.3; load test against target concurrent-match volume.
- **Completion criteria:** tracking accuracy and latency both meet documented per-tier targets on real (non-synthetic) match footage from at least 3 different matches.

### Phase 3 Exit Gate (new — CTO review)

Phase 4 does not begin until the measured accessible-tier accuracy ceiling is known and documented,
replacing the provisional ≥92% hypothesis from `prd.md` Section 11 with a real number. If the measured
ceiling is materially below a usable bar for umpiring decisions:
- **Pivot option A:** revise the accessible-tier hardware baseline (e.g., mandate 2 cameras minimum
  instead of 1) and re-test before proceeding.
- **Pivot option B:** narrow accessible-tier MVP scope to the decision types where monocular tracking is
  demonstrably sufficient (e.g., run-out/stumping, which depend less on 3D trajectory precision than
  LBW), deferring LBW to a 2-camera-minimum configuration.
- This gate exists specifically so a Phase 3 accuracy shortfall changes the roadmap, rather than being
  quietly absorbed and discovered again (worse) at Phase 5 or Phase 9.

---

## Phase 4 — Bat Detection & Edge Detection

**Objective:** Deliver the audio/visual fusion edge-detection capability for caught-behind and LBW-overturn scenarios.

- **Features delivered:** Edge/Impact Detection Service.
- **Technical tasks:**
  - Audio waveform spike-detection model tuned for bat-ball contact signature vs. ambient/crowd noise.
  - Visual micro-motion/deformation detection near the bat at point of delivery.
  - Fusion model combining both modalities into a calibrated confidence score.
  - Integration with Review Orchestration Service decision-package rendering.
- **Required skills:** ML engineer (multi-modal fusion experience preferred), audio signal processing specialist (can be a consulting/part-time role), CV engineer.
- **Dependencies:** Phase 3 tracking pipeline (shared infrastructure); labeled edge/no-edge dataset (extends Phase 0/3 data collection).
- **Deliverables:** Edge Detection Service deployed to staging with documented accuracy against a labeled benchmark.
- **Testing requirements:** false-positive/false-negative rate specifically benchmarked (edge decisions are historically the most contentious in real cricket, so this needs rigorous validation before any live use).
- **Completion criteria:** edge-detection accuracy meets a documented threshold agreed with cricket domain experts (leverage the product owner's own playing/umpiring-adjacent expertise here) on real match footage.

---

## Phase 5 — LBW Prediction Engine

**Objective:** Deliver the full LBW decision engine (pitching, impact, wicket-projection) combining physics modeling with the ML residual correction approach from `architecture.md` Section 8.

- **Features delivered:** LBW Trajectory Engine, decision-package visualization (trajectory replay graphic).
- **Technical tasks:**
  - Physics-based projectile/swing/seam model implementation.
  - ML residual correction model trained on tracked trajectory data from Phase 3.
  - Visualization rendering pipeline (trajectory + impact point + projected path graphic).
  - Rules engine for LBW-specific playing conditions (pitching outside leg, impact outside off with no shot, height variations, etc. — configurable per league's playing conditions).
- **Required skills:** ML engineer, physics/simulation-literate engineer, frontend engineer (visualization), cricket domain SME.
- **Dependencies:** Phase 3 (trajectory data, and the Phase 3 Exit Gate's validated accuracy ceiling — LBW is the decision type most sensitive to 3D trajectory precision, so this dependency is load-bearing, not incidental), Phase 4 (shared confidence/QA infrastructure).
- **Deliverables:** end-to-end LBW review flow demoable on real recorded deliveries with visual replay output.
- **Testing requirements:** validated against a panel of experienced umpires reviewing the same deliveries manually, against the **measured** accessible-tier ceiling established in Phase 3 (superseding the provisional ≥92% hypothesis in `prd.md` Section 11).
- **Completion criteria:** LBW engine passes the expert-panel validation threshold and end-to-end latency SLA on real match data.

### Phase 5 Exit Gate (new — CTO review)

Before Phase 7 integrates the LBW engine into the live umpire-facing product: if expert-panel agreement
falls meaningfully short of the Phase 3-validated ceiling (i.e., the LBW-specific model underperforms the
general tracking accuracy it's built on), treat this as a signal to invest further in the physics/ML
residual model before proceeding, rather than shipping a known-weak decision type into Phase 7's polished
UI, where confidence in the "evidence" presentation could outpace the model's actual reliability.

---

## Phase 6 — Run-out and Stumping Detection

**Objective:** Deliver frame-accurate run-out/stumping analysis.

- **Features delivered:** Run-out/Stumping Service.
- **Technical tasks:**
  - Bail-dislodgement detection (high-frame-rate frame differencing / lightweight motion model).
  - Batter/fielder pose estimation for foot/bat-vs-crease-line determination.
  - Frame-accurate timestamp correlation between bail dislodgement and body-part position.
  - Visualization: frame-by-frame zoom/line-overlay replay.
- **Required skills:** CV engineer (pose estimation experience), frontend engineer (frame-scrub visualization UI).
- **Dependencies:** Phase 2 ingestion (needs high-frame-rate capture where available), shared tracking infrastructure from Phase 3.
- **Deliverables:** working run-out/stumping review flow validated on real recorded dismissals and close-but-safe scenarios.
- **Testing requirements:** accuracy validated on a labeled set of real run-out/stumping footage, including genuinely close calls (the highest-value, highest-difficulty case).
- **Completion criteria:** frame-accuracy timestamp precision meets a documented tolerance (e.g., within N milliseconds) validated against known-ground-truth footage.

---

## Phase 7 — Review System and Visualization

**Objective:** Bring all decision engines together into the complete, polished Umpire Review Console experience and end-to-end review workflow.

- **Features delivered:** full Review Orchestration Service, Umpire Review Console (production UI per `design.md`), review-quota enforcement, immutable audit logging.
- **Technical tasks:**
  - Review workflow state machine (trigger → evidence assembly → human confirmation → persistence → broadcast).
  - Umpire Review Console UI build-out (per design system).
  - Immutable decision-record persistence and audit trail.
  - Review-quota/playing-conditions configuration per `prd.md` FR7.
  - Notification service integration (scoreboard, officials, connected apps).
- **Required skills:** full-stack engineer, frontend/UX engineer, backend engineer.
- **Dependencies:** Phases 3–6 (all decision engines available to integrate).
- **Deliverables:** a complete, demoable, end-to-end review experience usable in a live scrimmage/practice match setting.
- **Testing requirements:** full E2E test suite (Playwright) per `rules.md` Section 6.2; usability testing with real umpires (a genuine dogfood opportunity given the product owner's league connections).
- **Completion criteria:** successful pilot review sessions run during a real or simulated Montreal Overseas CC / MCA match with umpire feedback incorporated.

---

## Phase 8 — AI Model Improvement

**Objective:** Systematize the human-validation feedback loop and close the accuracy gap identified in field pilots; begin broadcast-tier model configuration work.

- **Features delivered:** model retraining pipeline fed by human-override data, broadcast-tier (multi-camera, high-frame-rate) model configuration and validation.
- **Technical tasks:**
  - Automated data pipeline capturing umpire override events as new labeled training data (with governance controls per `rules.md` Section 5.1).
  - Retraining/evaluation pipeline with regression testing against historical benchmark sets.
  - Broadcast-tier-specific model tuning (higher camera count, higher frame rate) and dedicated accuracy validation.
  - Expanded adversarial/edge-case test set (lighting, pink ball, kit color variation).
- **Required skills:** MLOps engineer, ML engineer, data engineer.
- **Dependencies:** Phase 7 pilot data; sufficient volume of real match reviews collected.
- **Deliverables:** documented, repeatable model-improvement pipeline; broadcast-tier accuracy report.
- **Testing requirements:** every model promotion follows the validation process in `rules.md` Section 6.3, with no exceptions.
- **Completion criteria:** measurable accuracy improvement (documented, versioned) over Phase 5–6 baseline models; broadcast-tier model meets its stricter accuracy target.

---

## Phase 9 — Production Deployment

**Objective:** Harden the platform for real, unsupervised production use across multiple venues/leagues.

- **Features delivered:** production-grade reliability, security hardening, multi-venue/multi-tenant operational readiness, board/organizer admin tooling, analytics dashboards, dispute/appeal workflow, finalized liability/ToS.
- **Technical tasks:**
  - Load testing and capacity planning for concurrent-match volume (`rules.md` Section 6.4).
  - **Full penetration test and security audit** (`rules.md` Section 6.5) — note this builds on tenant-isolation and secrets-management testing that has been continuous since Phase 1, not a first-time security pass.
  - Runbook completion for all identified failure modes (`rules.md` Section 4.4).
  - Analytics & Reporting Service full build-out (player/coach/board dashboards) — **on the Phase 8 warehouse**, not the Phase 7 direct-OLTP queries (`architecture.md` Section 12).
  - Disaster recovery and backup procedures validated.
  - **(New — CTO review) Post-match dispute/appeal workflow** (`prd.md` Section 5.7) implemented and available to board admins.
  - **(New — CTO review) Liability/ToS and privacy compliance finalized** (`prd.md` Section 13), building on the Phase 1 legal-review kickoff — this must be complete, not in-progress, before any real paying league goes live.
  - **(New — CTO review) Directional cost model validated against actuals** (`architecture.md` Section 19) from real production usage, informing Phase 10 pricing.
- **Required skills:** SRE/platform engineer, security engineer, backend/frontend engineers for analytics build-out, legal/privacy counsel (final sign-off).
- **Dependencies:** Phases 1–8 complete and pilot-validated.
- **Deliverables:** production environment live, first paying/real league (e.g., MCA) fully onboarded and running live matches.
- **Testing requirements:** full security audit passed; load test results meeting documented SLAs; DR failover tested at least once.
- **Completion criteria:** at least one full season or tournament run successfully in production with acceptable accuracy, uptime, and umpire/organizer satisfaction metrics.

### Phase 9 Exit Gate (new — CTO review)

Phase 10 commercialization does not begin until: (1) actual production cost-per-match is known and
compared against Phase 0's business-model validation findings — if the gap between real cost and
realistic willingness-to-pay hasn't closed, that's a pricing/scope problem to solve before scaling
outreach, not after; and (2) legal/liability sign-off is complete for every jurisdiction in active use.

---

## Phase 10 — Commercialization and Scaling

**Objective:** Scale beyond the initial league(s), formalize pricing/tiering, and pursue broadcast/partner integrations.

- **Features delivered:** self-serve organizer onboarding, billing/subscription system, broadcast overlay partner integrations, certified-installer program for venue calibration.
- **Technical tasks:**
  - Billing/subscription infrastructure (per-match/per-season pricing per `prd.md` persona needs).
  - Multi-region deployment for latency and data-residency as adoption grows geographically.
  - Broadcast overlay integration work per partner (acknowledging this is bespoke per `architecture.md` Section 18 risk).
  - Fan-facing features and monetization surfaces (Phase-gated per `prd.md` Section 12).
- **Required skills:** growth/product engineer, partnerships-focused solutions engineer, continued platform/SRE support.
- **Dependencies:** Phase 9 successful production track record as a reference case.
- **Deliverables:** documented onboarding path for new leagues; at least one broadcast/streaming partner integration live.
- **Testing requirements:** billing system tested for correctness/edge cases (proration, refunds); partner integration tested against each partner's specific graphics pipeline.
- **Completion criteria:** platform actively serving multiple independent leagues/organizations beyond the founding one, with sustainable unit economics per match/season.

---

## Cross-Phase Notes

- **Data collection is continuous**, not confined to Phase 0 — every phase from 2 onward should be capturing real footage and (where consented) labeled data, since model quality is the product's core differentiator.
- **Domain-expert validation (umpires, experienced players)** should be looped in from Phase 3 onward, not saved for a "UAT phase" at the end — this is standard practice for AI systems with high real-world consequence, and directly leverages the product owner's own domain fluency.
- **Broadcast tier is deliberately deferred** to Phase 8+ to avoid over-building for a smaller, harder, more capital-intensive segment before the core technology is proven in the accessible-tier market.
