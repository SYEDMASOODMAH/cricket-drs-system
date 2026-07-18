# Product Requirements Document (PRD)
## AI-Powered Cricket Decision Review System (Cricket DRS)

**Status:** Draft v1.1 — updated post CTO review
**Owner:** Product/Architecture
**Last updated:** 2026-07-18

**Changelog (v1.0 → v1.1):** incorporates the Critical/High findings from the pre-Phase-1 CTO review —
added a Legal, Privacy & Consent Framework (Section 13), a dispute/appeal workflow, explicit hardware/cost
assumptions for accessible tier, revised accuracy targets to "validated, not asserted," and moved business-model
validation earlier (Phase 0/1, see `phases.md`).

---

## 1. Product Overview

Cricket DRS is an AI-powered decision review platform that uses computer vision, sensor fusion, and machine learning to deliver fast, accurate, and explainable umpiring decisions — LBW, edge detection (bat/pad), run-outs, stumpings, and boundary/catch validation — from standard or multi-camera video feeds.

The system is designed to scale from **grassroots/club cricket** (single camera, phone-based) up to **professional/broadcast-grade** deployments (multi-camera, Hawk-Eye-class tracking, ball-tracking radar/optical hybrid), sharing one core AI/architecture platform with tiered capability.

This is a meaningful design fork from the outset: traditional DRS (Hawk-Eye, Virtual Eye) is built for broadcast production with dedicated hardware and six+ synchronized high-frame-rate cameras. A large, underserved market — club/league cricket like the Montreal Cricket Association — has no access to DRS at all. The PRD treats **"broadcast tier"** and **"accessible tier"** as two product configurations of one platform, not two products.

---

## 2. Vision and Mission

**Vision:** Every cricket match, from a Saturday club game to an international final, deserves access to fair, fast, and transparent decision-making.

**Mission:** Democratize ball-tracking and decision-review technology through AI, making it affordable, accurate, and explainable — while providing professional-grade tooling for boards, franchises, and broadcasters who need certification-level accuracy.

---

## 3. Problem Statement

- Traditional DRS (Hawk-Eye, Virtual Eye) costs **$5,000–$15,000+ per match** in hardware, personnel, and licensing — inaccessible to club, school, and domestic cricket.
- Officiating errors in non-broadcast cricket are common and unappealable, causing disputes, safety issues (in the case of dangerous bowling), and loss of trust in leagues.
- Existing consumer apps (scoring apps, basic camera replay) provide **no trajectory prediction, no edge detection, and no LBW simulation** — they are recording tools, not decision tools.
- Umpires at all levels lack real-time analytical support (e.g., objective snick detection), forcing high-stakes decisions on human perception alone, at normal camera frame rates.
- There is no unified platform that lets a club league today, and a semi-pro or franchise league tomorrow, grow on the same product without switching vendors.

---

## 4. Target Users / Personas

### 4.1 Cricket Players
- Want fair decisions and, secondarily, personal performance analytics (dismissal patterns, review success rate).
- Pain point: perception that "close calls" go against them repeatedly without recourse.

### 4.2 Coaches
- Need aggregate and individual analytics: LBW dismissal trends, bowling line/length heatmaps, batting weaknesses against particular deliveries.
- Use DRS review data as a coaching and opposition-scouting tool, not just an in-match arbitration tool.

### 4.3 Umpires
- Primary decision-makers; need a **decision support tool**, not a replacement. The system must present evidence (trajectory, impact point, ball-tracking confidence) and leave final authority with the human umpire (see `rules.md` — AI Boundaries).
- Need a low-friction review workflow: trigger review, view result, communicate decision — ideally under 60–90 seconds even at accessible tier.

### 4.4 Tournament Organizers
- Need scheduling, review-quota management (e.g., "2 unsuccessful reviews per innings"), match configuration (single vs multi-camera), and post-match report generation.
- Cost-sensitivity is highest in this persona — pricing/tiering must reflect a per-match or per-season model, not enterprise SaaS pricing only.

### 4.5 Cricket Boards / Leagues
- Need governance: audit trails of every review decision, historical accuracy reporting, compliance with ICC-style playing conditions, and the ability to certify/calibrate camera rigs per venue.
- Multi-tenant administration across many affiliated clubs/venues.

### 4.6 Fans (secondary, phase-gated)
- Want the "TV-style" DRS graphic experience (ball trajectory replay, snickometer-style audio-visual, hotspot-style overlays) for engagement, even at club level, via live-stream overlays.
- Monetization surface (see Section 12) but not core MVP.

---

## 5. User Journeys

### 5.1 Umpire-Initiated Review (core flow)
1. On-field umpire signals a review (via referee device/app or verbal + third-umpire trigger).
2. System ingests the last N seconds of synchronized video from the last-triggered camera set.
3. AI pipeline runs ball detection → tracking → event classification (pitch, impact, wicket) → decision-type-specific model (LBW physics engine, edge-detection audio/visual fusion, run-out frame analysis).
4. Result rendered as a visual replay + confidence score + decision recommendation within a target SLA (tier-dependent, see NFRs).
5. Third umpire/human reviewer confirms or overrides; final decision is logged immutably.
6. Decision and full evidence trail pushed to scoreboard, broadcast overlay (if enabled), and match record.

### 5.2 Player Reviewing Personal Stats (post-match)
1. Player logs into mobile/web app.
2. Views match history → selects match → sees own dismissals with DRS replay clips and dismissal-type breakdown.
3. Views season-level analytics (dismissal types, review success rate if they requested reviews).

### 5.3 Coach Building a Scouting Report
1. Coach selects opposition team.
2. System aggregates historical ball-by-ball + dismissal + trajectory data across all DRS-recorded matches for that team/players.
3. Coach exports a scouting report (PDF/HTML) with heatmaps, dismissal tendencies, and video clip links.

### 5.4 Tournament Organizer Setting Up a Match
1. Organizer creates a match in the admin console, selects venue (pre-calibrated camera rig or ad-hoc phone setup), assigns officials, sets review-quota rules.
2. **(Revised — hardware model update)** Organizer or a designated technician physically mounts the platform-provided GoPro kit at pre-calibrated tripod/stump-mount positions and connects each camera via USB-C tether to the venue edge box.
3. System runs a pre-match camera detection and calibration/health check — confirming each registered camera is connected, seeing the pitch correctly, and reporting expected frame rate; also measures uplink bandwidth to select cloud vs. local-inference mode for the match (`architecture.md` Section 9a).
4. Edge box begins rolling-buffer capture per camera; match goes live; organizer monitors a live camera/system-status indicator alongside the review usage dashboard.

### 5.5 Board Auditing Season Accuracy
1. Board admin opens governance dashboard.
2. Filters by league/season/umpire.
3. Reviews AI-vs-human-override rate, average decision confidence, and any decisions flagged for post-hoc dispute.

### 5.6 Player/Club Onboarding Consent (new — CTO review)
1. Before a club or player is added to any match roster, the organizer or club admin captures explicit consent for video capture, AI analysis, and (where applicable) footage reuse in highlights/scouting features.
2. Consent status is stored per player and is a precondition for that player appearing in any DRS-enabled match — the system must not silently record and analyze a non-consenting player.
3. Players/guardians (for minors, where applicable) can review and revoke consent at any time via the mobile/web app; revocation is honored for all future matches (see Section 13).

### 5.7 Post-Match Dispute / Appeal (new — CTO review)
1. A team, player, or club submits a post-match dispute against a human-confirmed decision (distinct from disagreeing with the AI recommendation itself, which is resolved in-match via override).
2. Organizer/board reviews the full evidence package (video, AI decision package, confidence score, human reviewer identity and timestamp) already retained per the immutable audit log.
3. Board issues a ruling per league playing conditions; the ruling and rationale are appended to the immutable record — the original decision is never altered, only annotated with the appeal outcome (preserving audit integrity).

---

## 6. Core Features (MVP)

- Multi-tier video ingestion: single phone camera (accessible) up to 6+ synchronized cameras (broadcast).
- Ball detection and 2D/3D trajectory tracking.
- LBW decision engine (pitching, impact, wicket-projection) with confidence scoring.
- Bat/pad edge detection (audio spike + visual micro-motion fusion) for caught-behind/LBW-overturn scenarios.
- Run-out / stumping frame-accurate bail-and-foot/bat analysis.
- Review workflow UI for on-field/third umpire with evidence visualization.
- Match, team, player, and venue management (multi-tenant).
- Review-quota and playing-conditions configuration per league/tournament.
- Immutable audit log of every review and decision.
- Player/coach analytics dashboards (post-match).
- Role-based access control (player, coach, umpire, organizer, board admin, fan).
- Player/club consent capture and management for video recording and AI analysis (new — CTO review; see Section 13).
- Post-match dispute/appeal workflow with append-only ruling annotation (new — CTO review; see Section 5.7).

## 7. Advanced Features (Post-MVP)

- Broadcast-grade real-time overlay generation (trajectory graphics, "hotspot"-style thermal-like edge visualization) for livestream integration.
- Predictive analytics: bowler-specific LBW success probability, batter dismissal-risk heatmaps.
- Automated highlights generation (wickets, boundaries, close calls) using event detection.
- Multi-venue camera calibration marketplace (certified installer network).
- Live radar-gun-style speed and swing/seam analytics.
- Historical "what-if" replays (re-running an old decision against updated models).
- Umpire performance analytics and training mode (practice reviews on historical footage).
- Fan-facing companion app with prediction games tied to live reviews.

## 8. AI-Powered Capabilities

- **Ball detection & tracking model**: object detection (YOLO-class) + trajectory Kalman/physics-informed filtering, tuned for small fast-moving objects at variable frame rates.
- **Trajectory prediction (LBW)**: physics-based projectile model (accounting for swing, spin-induced drift, seam effects) calibrated/corrected by ML residual model trained on tracked trajectories.
- **Edge/impact detection**: multi-modal fusion of audio waveform spike detection (bat-on-ball) with visual micro-deformation/motion analysis near the bat, producing a confidence-scored "edge/no-edge" classification.
- **Pose estimation**: batter/fielder/wicket pose estimation to support run-out (foot/bat vs crease) and stumping (bail dislodgement timing vs foot position) analysis.
- **Confidence scoring & uncertainty quantification**: every AI decision ships with a calibrated confidence interval, not just a binary output — critical for human-in-the-loop trust (see `rules.md`).
- **Camera calibration assist**: AI-assisted auto-calibration for ad-hoc (non-fixed-rig) camera setups using known pitch geometry (22-yard length, stump dimensions) as reference.
- **Anomaly/QA model**: flags low-confidence or physically implausible tracking results (e.g., occluded ball) for mandatory human review rather than silently guessing.

## 9. Functional Requirements

- FR1: System shall ingest video from 1–8 synchronized camera sources per match.
- FR2: System shall detect and track the ball at ≥95% frame-detection rate in unobstructed conditions (broadcast tier), ≥85% (accessible tier).
- FR3: System shall generate an LBW decision (out/not-out + trajectory visualization) within tier-defined SLA.
- FR4: System shall detect bat-ball contact (edge) with a confidence score and supporting audio/visual evidence.
- FR5: System shall support run-out/stumping frame-by-frame review with bail-dislodgement and body-part-vs-line timestamping.
- FR6: System shall log every review request, evidence set, AI output, and human-final-decision immutably, with timestamp and reviewer identity.
- FR7: System shall enforce configurable review-quota rules per league/playing conditions.
- FR8: System shall support role-based multi-tenant access across clubs, leagues, and boards.
- FR9: System shall expose a public/partner API for scoring apps and broadcast graphics systems to consume decision data.
- FR10: System shall generate post-match analytics reports per player, team, and match.

## 10. Non-Functional Requirements

- **Latency (broadcast tier):** end-to-end review decision in ≤ 30 seconds (matching real-world DRS SLAs).
- **Latency (accessible tier):** end-to-end review decision in ≤ 90 seconds **measured against a defined minimum venue uplink bandwidth** (see `architecture.md` Section 9a). **CTO review note:** the original NFR assumed reliable venue-to-cloud upload while the target market (Section 3) is specifically venues *least* likely to have it — this is now resolved architecturally via a local/edge-inference fallback path for below-minimum-bandwidth venues, with a wider, disclosed SLA (target ≤ 3 minutes) in that fallback mode rather than a silent failure or an SLA nobody can actually hit.
- **Availability:** 99.9% for live-match-critical services during scheduled match windows; graceful degradation to "manual review, no AI assist" rather than hard failure. Degradation must also cover **partial/corrupted evidence** (e.g., a dropped camera feed mid-clip), not only total AI-pipeline unavailability — the system must detect and flag a partial evidence set rather than presenting it as if complete (new — CTO review).
- **Accuracy:** ball-tracking spatial error ≤ 3mm at stump line (broadcast tier, multi-camera) — aligned to ICC's Ball Tracking Standard tolerance; accessible-tier accuracy target is **provisional pending Phase 0 validation** (see Section 11 and `phases.md` Phase 0 exit criteria) rather than asserted up front, given the physics-limited ceiling of monocular/dual-camera tracking.
- **Auditability:** all decisions must be reconstructable and explainable after the fact (evidence + model version + confidence); decision records use a cryptographic hash-chaining scheme (not append-only storage alone) so a compromised credential cannot forge a *replacement* record without detection (see `architecture.md` Section 15).
- **Security:** all match video and personal data encrypted at rest and in transit; strict tenant isolation for board/league data, with isolation testing performed continuously from Phase 1 onward, not deferred to pre-production hardening (see `rules.md` Section 6.5).
- **Scalability:** platform must support concurrent multi-match live processing (target: 50 simultaneous matches at accessible tier per region within 18 months), subject to the directional GPU/infrastructure cost model in `architecture.md` Section 19.
- **Portability (revised — hardware model update):** accessible tier ships as a **platform-provided hardware kit** (bundled with the subscription), not "bring your own phone." **Reference kit (new):** 2–4 GoPro-class action cameras (120fps+ capable, e.g., Hero 11/12/13 tier) per venue, mounted on fixed tripods/stump-mounts, connected via **USB-C tethered capture** (webcam mode or HDMI+capture-card fallback) to the venue edge box — wired, not WiFi, to avoid the reliability risk of GoPro's native wireless streaming. No proprietary *cloud* lock-in (the platform, not the camera vendor, owns the pipeline), but camera hardware is now a defined, platform-controlled reference kit rather than an arbitrary consumer device. This is a deliberate shift from the original "any commodity smartphone" assumption — see Section 14 for rationale.
- **Explainability:** every AI decision must include a human-readable rationale, not just a score, to preserve umpire and player trust.
- **Privacy & jurisdictional compliance (new — CTO review):** the platform must comply with applicable privacy law in each operating jurisdiction, including Quebec's *Law 25* and Canada's *PIPEDA* for the initial market, given that pose-estimation-derived data on identifiable players is privacy-sensitive. See Section 13.
- **Integrity (new — CTO review):** camera feeds and the review pipeline must be resistant to reasonably foreseeable tampering (lens obstruction, feed substitution, replay injection); see Section 13.4 and `rules.md` Section 6.5.

## 11. Success Metrics

- Decision accuracy validated against expert human panel review. Broadcast tier targets ≥ 98% agreement (established DRS industry benchmark). **Accessible-tier target is provisional** — Phase 0 must produce a real, measured accuracy ceiling for 1–2 camera monocular/dual-camera tracking before a committed target is set; ≥ 92% is a working hypothesis to validate, not a guaranteed outcome, given the physics constraints of limited-camera setups (see `phases.md` Phase 0 exit criteria — new, CTO review).
- Average review turnaround time within SLA in ≥ 95% of reviews.
- Adoption: number of leagues/clubs onboarded, matches processed per month.
- Umpire trust score (survey-based) and override rate trending down over time as model confidence calibration improves.
- Reduction in post-match dispute reports for DRS-enabled matches vs non-DRS matches (self-reported by leagues).
- Player/coach engagement: % of matches where analytics dashboards are viewed within 48 hours.

## 12. Future Expansion Opportunities

- Full broadcast partnership integrations (graphics packages for streaming platforms).
- Expansion to other bat-and-ball / LBW-adjacent sports and umpiring use cases where the underlying trajectory/impact modeling generalizes.
- Wearable sensor fusion (smart bails, embedded ball chips) as an alternative/complementary sensing modality to pure computer vision.
- AI-assisted coaching recommendations beyond DRS (bowling plans, field placement optimization).
- Marketplace for certified venue installers and camera rig rental for club-level leagues.
- Fan engagement monetization: pay-per-view close-call replays, prediction contests, sponsorship on review overlays.

---

## 13. Legal, Privacy & Consent Framework (new — CTO review, Critical)

This section did not exist in v1.0 and is added because the product captures and analyzes video of
identifiable people (including pose-estimation-derived data, which is privacy-sensitive in several
jurisdictions) and makes decisions with real competitive and reputational consequences.

### 13.1 Liability
- The product is a **decision-support tool**; the human umpire's confirmed decision, not the AI
  recommendation, is the official match decision (reinforced by the AI-boundary rule in `rules.md`
  Section 5.1). Terms of service must state this explicitly to every league/organizer at onboarding.
- Liability for a wrong human-confirmed decision rests with the league's existing playing-conditions
  and dispute process (Section 5.7), not with the platform. Liability for a **system fault** (e.g., the
  platform presents corrupted evidence as valid) is a genuine open question requiring legal review
  (insurance/indemnification terms) before any commercial contract — flagged here, not resolved here.

### 13.2 Privacy & Jurisdictional Compliance
- Initial market (Quebec/Canada) requires compliance with **Law 25** (Quebec private-sector privacy law,
  which has specific requirements for biometric-adjacent data) and **PIPEDA** at the federal level.
  Pose-estimation output on identifiable players is treated as sensitive personal data, not generic
  telemetry.
- Data retention/deletion policy (referenced in `architecture.md` Section 15) must be published to
  players/clubs, not just implemented internally, and must support a deletion request workflow.
- Expansion to new jurisdictions requires a jurisdiction-specific privacy review before launch — this is
  a recurring checklist item in `phases.md` Phase 10, not a one-time Phase 1 task.

### 13.3 Consent Model
- No player's video/pose data is captured or analyzed without prior consent captured through the
  onboarding journey in Section 5.6. Consent is per-player, revocable, and covers: video capture, AI
  analysis, and (separately, opt-in) footage reuse in highlights/scouting/marketing.
- Minors require guardian consent; this must be a distinct, explicit flow, not an assumption folded
  into general club registration.

### 13.4 Integrity / Anti-Tampering
- The system must be resistant to reasonably foreseeable manipulation: deliberate lens obstruction,
  camera feed substitution, or replay/stream injection. Minimum controls: signed/authenticated camera
  feed sessions, server-side validation that received frames are live (not replayed) footage, and
  anomaly alerts when a camera's feed characteristics change unexpectedly mid-match. Full threat model
  owned by `rules.md` Section 6.5.

### 13.5 Hardware Kit Provisioning (new — hardware model update)
- Each physical camera in the platform-provided GoPro kit is registered with a unique ID tied to a
  club/venue at onboarding (Section 5, Stage A of the setup flow). This is what lets the Camera
  Calibration Service reuse a saved calibration profile per camera/venue across matches instead of
  recalibrating from scratch every time.
- Kit ownership sits with the platform (bundled into the subscription), not the club — this is a
  deliberate business-model choice (Section 14 assumptions) that also simplifies hardware consistency
  for the ML pipeline (one known lens/frame-rate profile instead of arbitrary consumer devices).
- Kit maintenance/replacement (battery degradation, physical damage) is a subscription-service
  responsibility and should be scoped into Phase 10 pricing, not treated as a one-time capital cost.

---

## 14. Key Assumptions (documented per instructions)

- Assumption: MVP targets accessible-tier (1–2 camera, club/league) first, since it is the underserved, faster-to-validate market; broadcast-tier is a Phase 8+ capability once core CV/ML pipeline is proven.
- Assumption: The system augments, never replaces, human umpires — this is a hard product and ethical boundary (see `rules.md`).
- Assumption: Regulatory/certification requirements (ICC-style approval for professional matches) are out of scope for MVP and treated as a later compliance milestone.
- Assumption: Initial geographic focus is leagues similar to the Montreal Cricket Association (organized club cricket, semi-formal video infrastructure) before scaling to franchise/professional cricket.
- **Assumption (revised — CTO review): business-model/willingness-to-pay validation happens early** (Phase 0/1, via direct conversation with MCA and comparable leagues), not deferred to Phase 10 — the original sequencing risked eight-plus phases of investment before testing whether the target market will actually pay.
- **Assumption (revised — hardware model update):** accessible-tier hardware is a **platform-provided GoPro-class camera kit, bundled with the subscription**, not club/venue-owned or rented — superseding the earlier "club/venue owns or rents commodity hardware" assumption. This directly supports the subscription business model (hardware-as-a-service) and must still be validated against real MCA-tier club budgets and willingness-to-pay in Phase 0/1, now specifically for a subscription-inclusive-of-hardware price point rather than a software-only price point.
