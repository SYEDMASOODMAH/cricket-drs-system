# Design System
## AI-Powered Cricket Decision Review System (Cricket DRS)

**Status:** Draft v1.0
**Companion to:** `prd.md`, `architecture.md`

---

## 1. Product Branding Direction

The brand should read as **precision sports technology**, not a consumer scoring app and not broadcast kitsch. The reference bar is deliberately high: Apple's material restraint, Tesla's confident minimalism, and the clean data-density of platforms like Hawk-Eye's own graphics or modern F1/analytics broadcast packages. The product is making high-stakes calls in front of players and crowds — the visual language must communicate **authority, clarity, and calm precision** even under pressure.

**Brand personality:** precise, trustworthy, fast, understated-premium, technical but never cold.

**Avoid:** generic "sports app" clichés (loud gradients, aggressive angular shapes, stock-photo athletes, cluttered dashboards). Avoid anything that looks like a mobile game.

---

## 2. Design Philosophy

- **Evidence over decoration.** The trajectory visualization, confidence score, and replay are the product. Chrome, navigation, and branding should recede so evidence is always the visual focus, especially in the Umpire Console.
- **Calm under pressure.** Review moments are high-stress (players waiting, crowd watching). The UI must never feel chaotic — generous whitespace, restrained motion, clear hierarchy, no unnecessary animation competing for attention during a live decision.
- **Progressive density.** Umpire Console = minimal, single-focus, big-touch-target. Coach/Board analytics = data-dense, information-rich, built for exploration. Same design language, different density profile — not different products.
- **Trust through transparency.** Confidence scores, rationale, and evidence are always visible, never hidden behind a "just trust the AI" black box — this is a UX expression of the AI-boundary principle in `rules.md` Section 5.

---

## 3. Color Palette

**Primary palette — "Pitch & Precision":**

| Role | Color | Hex (indicative) | Usage |
|---|---|---|---|
| Primary (brand) | Deep Cricket Green | `#0B3D2E` | Primary brand color, headers, key actions |
| Accent (precision) | Signal Amber | `#F5A623` | Ball-tracking trajectory line, active review indicator |
| Accent (confidence-high) | Confidence Green | `#2E9E5B` | High-confidence decisions, "out"/"not out" positive states |
| Accent (confidence-low/caution) | Caution Amber-Red | `#D9642C` | Low-confidence flags, requires-manual-review states |
| Decision — Out | Decision Red | `#C0392B` | "Out" decision state (used sparingly, high-signal only) |
| Decision — Not Out | Decision Blue | `#2C6BC0` | "Not out" decision state |
| Neutral base (dark theme) | Charcoal | `#12161A` | Primary background, Umpire Console default |
| Neutral base (light theme) | Off-White | `#F7F8F6` | Primary background, Web/Mobile default |
| Text (dark theme) | Warm White | `#EDEFEC` | Primary text on dark backgrounds |
| Text (light theme) | Near-Black | `#171A1C` | Primary text on light backgrounds |

**Rationale:** deep green ties to the pitch/turf without being a literal "grass green" cliché; amber is reserved specifically for the trajectory/tracking visual language so it reads consistently as "this is AI-tracked data" wherever it appears. Red/blue for out/not-out draw on universal, immediately legible sports conventions (avoids ambiguity in a high-stakes moment).

---

## 4. Dark/Light Theme Recommendations

- **Umpire Review Console: dark theme by default, no light-theme toggle initially.** Rationale: pitch-side/stadium and broadcast-truck viewing conditions favor a dark UI (reduces glare, improves outdoor screen legibility for evidence-critical viewing, matches broadcast monitoring environments).
- **Web App (Coach/Organizer/Board) and Mobile App: light theme by default, with a dark theme option.** These are analytics/admin contexts used indoors/at a desk as often as outdoors; user choice matters more than a single prescribed environment.
- Both themes share the same semantic color tokens (confidence, decision-state colors) so a screenshot/replay looks recognizable regardless of which surface it's viewed on.

---

## 5. Typography

- **Primary typeface:** a high-legibility, modern grotesque/geometric sans-serif — e.g., **Inter** (open-source, excellent numeral legibility critical for scores/timestamps/confidence percentages, extensive weight range).
- **Numerals/data typeface:** tabular-figure variant of the same family for all scores, speeds, percentages, and timestamps, ensuring columns of numbers align cleanly in dashboards.
- **Type scale (indicative, 8-step modular scale):**
  - Display (match/decision headline): 32–40px, weight 600–700
  - H1 (section headers): 24–28px, weight 600
  - H2 (subsection): 18–20px, weight 600
  - Body: 15–16px, weight 400
  - Caption/metadata: 12–13px, weight 400–500
  - Confidence/decision badge: 14–16px, weight 700, uses tabular numerals

**Rationale:** cricket decision UI lives or dies on numeral clarity (confidence %, speeds, timestamps) — typeface choice is functional, not just aesthetic.

---

## 6. Spacing System

- **8px base unit grid** (4px used only for micro-adjustments within components), consistent across web, mobile, and console — ensures the shared component library (per `architecture.md` Section 6) behaves predictably across surfaces.
- Standard spacing scale: `4, 8, 12, 16, 24, 32, 48, 64, 96` (px), exposed as design tokens (`space-1` through `space-9`) consumed by both React (web/console) and Flutter (mobile) via a shared design-token source of truth (e.g., a Style Dictionary–generated token set).
- **Umpire Console specifically** uses larger-than-default spacing and touch targets (minimum 48x48px per accessibility guidance below) given tablet/pitch-side use under time pressure — precision tapping should never be a source of error in a high-stakes workflow.

---

## 7. Component Design Principles

- **Every component has a single, obvious purpose** — no overloaded multi-function components in the review path (directly mirrors the Single Responsibility principle from `rules.md`, applied to UI).
- **State is always visually explicit:** loading (AI processing), low-confidence (flagged), confirmed, overridden — each has a distinct, consistent visual treatment reused everywhere, not re-invented per screen.
- **Motion is purposeful, not decorative:** the trajectory-drawing animation on a review result is meaningful motion (it *is* the evidence); anything else (page transitions, hover states) is fast (150–200ms) and subtle.
- **Accessible by default**, not retrofitted (see Section 12).
- Shared component library (buttons, badges, cards, data tables, confidence indicators, trajectory-viewer shell) built once and themed per-surface, per the cross-platform design-token approach above.

---

## 8. Dashboard Design (Coach / Organizer / Board)

- **Layout:** left-nav persistent navigation (match/team/player/season selectors), main content area using a card-and-table hybrid grid — dense but never cluttered, generous internal card padding even in data-heavy views.
- **Hierarchy:** headline KPI cards (e.g., review accuracy rate, matches played, dismissal breakdown) above the fold; drill-down tables and charts below.
- **Charting:** consistent chart library and palette across all dashboards (trajectory-amber and decision red/blue reserved specifically for decision-related charts; a separate neutral analytical palette — blues/teals/grays — used for general statistics to avoid visual confusion between "this is a decision" and "this is a stat").
- **Board/governance dashboard specifically:** emphasizes trust/audit signals — override rates, confidence distributions, flagged-decision review status — laid out with the same clarity-first principle as the Umpire Console, since this is also a high-stakes trust surface.

---

## 9. Umpire Review Screen Design

This is the single most important screen in the product.

- **Layout:** full-screen, single-focus. Top third: match context (teams, over, batter/bowler, review type) in minimal text. Center: the trajectory/evidence visualization, largest element on screen by far. Bottom: confidence score + AI recommendation + two large, unambiguous action buttons (Confirm AI decision / Override).
- **No competing UI elements** during active review rendering — notifications, chat, and secondary navigation are suppressed while a review is in progress.
- **Confidence presentation:** a clear numeric percentage plus a plain-language rationale line (e.g., "Ball tracking confidence: 94% — pitching in line, impact in line, projected to hit leg stump"), never a bare score with no explanation, directly supporting the AI-boundary transparency principle.
- **Explicit low-confidence state:** when confidence falls below the review threshold, the screen visually and textually foregrounds "AI recommendation unavailable — manual review required" rather than showing a weak guess with a small disclaimer.
- **Timing pressure design:** a subtle, non-anxiety-inducing progress indicator during AI processing (target latency communicated in `architecture.md` Section 11), so umpires always know the system is working, never wondering if it's frozen.

---

## 10. Video Playback Interface

- **Frame-accurate scrubber** with clearly marked key-event frames (pitch point, impact point, bail dislodgement) as visual markers directly on the timeline — not just a plain progress bar.
- **Multi-camera angle switcher** (broadcast tier) presented as clear labeled thumbnails, not a hidden dropdown — angle comparison is often decision-relevant.
- **Slow-motion/frame-step controls** prominent and large (frame-back, frame-forward, play at 0.25x/0.5x/1x) given their centrality to the run-out/edge review use case.
- **Trajectory overlay toggle:** ability to show/hide the AI-rendered trajectory line on top of real video, letting the umpire compare raw footage against the AI interpretation directly — reinforces trust through transparency rather than replacing the raw evidence.

---

## 11. Analytics Dashboard Design (Player/Coach)

- **Player view:** personal, narrative-driven — "your season," dismissal-type breakdown as a simple donut/bar chart, recent match cards with replay links. Warmer, more approachable tone than the board/governance dashboards, while staying within the same core visual system.
- **Coach/scouting view:** comparison-first layout (player vs. player, team vs. opposition), heatmaps for bowling line/length and batting dismissal zones using a clear sequential color scale (avoid red/green-only scales for accessibility, per Section 12), exportable report generation prominent as a primary action.

---

## 12. Mobile Experience Guidelines

- Mobile (Flutter) app prioritizes **glanceable, notification-driven** interaction: match results, "you were reviewed — see the replay," season summaries — not an attempt to cram the full web dashboard into a small screen.
- Replay/trajectory visualization must remain the visual centerpiece on mobile too — never degrade to a static image; the animated trajectory replay is core brand-differentiating content.
- Offline-tolerant: cached recent match/replay data available without connectivity (relevant for club venues with poor signal), consistent with the edge-buffering resilience principle in `architecture.md`.
- Bottom-tab navigation (Home, Matches, Stats, Profile) kept to 4–5 items max; no hidden hamburger-menu burial of primary features.

---

## 13. Accessibility Standards

- **WCAG 2.1 AA compliance minimum** across all surfaces, AAA targeted for critical decision-state color contrast specifically (the out/not-out and confidence indicators must never rely on color alone — always paired with text/icon/pattern, addressing color-blindness directly given red/green/amber usage).
- Minimum touch target size 48x48px on all interactive elements, enforced strictly on the Umpire Console given its time-pressured, high-stakes usage context.
- Full keyboard navigability on web/console surfaces; screen-reader labeling on all data visualizations (trajectory charts, heatmaps) with a text-equivalent summary, not just a visual chart.
- Motion-reduction respect: honor `prefers-reduced-motion` for non-essential animation (page transitions, hover effects); the core trajectory-replay animation remains available but should offer a static-frame alternative view.
- Text scaling supported up to 200% without layout breakage across web and mobile surfaces.

---

## 14. Design System Governance

- Design tokens (color, spacing, typography) maintained as a single source of truth (Style Dictionary or equivalent) consumed by React (web/console) and Flutter (mobile) builds, preventing visual drift between platforms as the product scales per `phases.md`.
- Any new decision-state color or confidence-visualization pattern requires review against Section 13 accessibility standards before adoption — accessibility is a gate, not a follow-up pass.
- Component library versioned alongside the codebase (per `rules.md` documentation standards), with a living style-guide/Storybook reference kept current as the single design reference for engineers.
