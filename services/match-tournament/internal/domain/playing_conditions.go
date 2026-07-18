package domain

// DecisionType is one of the review types a match's playing conditions can
// enable, per prd.md Section 6/8.
type DecisionType string

const (
	DecisionTypeLBW      DecisionType = "lbw"
	DecisionTypeEdge     DecisionType = "edge"
	DecisionTypeRunOut   DecisionType = "run_out"
	DecisionTypeStumping DecisionType = "stumping"
)

func (d DecisionType) Valid() bool {
	switch d {
	case DecisionTypeLBW, DecisionTypeEdge, DecisionTypeRunOut, DecisionTypeStumping:
		return true
	default:
		return false
	}
}

// CameraTier matches architecture.md Section 1a's tier vocabulary exactly.
type CameraTier string

const (
	CameraTierAccessible CameraTier = "accessible"
	CameraTierBroadcast  CameraTier = "broadcast"
)

func (c CameraTier) Valid() bool {
	switch c {
	case CameraTierAccessible, CameraTierBroadcast:
		return true
	default:
		return false
	}
}

// PlayingConditions is the review-quota rule and decision-type
// configuration for a tournament or match, per prd.md Section 6/FR7. Phase
// 1 stores the rule only; enforcement (tracking reviews consumed per
// innings) is a Review Orchestration Service concern (phases.md Phase 7) —
// there are no review events yet to consume against.
type PlayingConditions struct {
	ReviewQuotaPerInnings int
	DecisionTypesEnabled  []DecisionType
	CameraTier            CameraTier
}

func NewPlayingConditions(reviewQuotaPerInnings int, decisionTypes []DecisionType, tier CameraTier) (PlayingConditions, error) {
	if reviewQuotaPerInnings < 0 {
		return PlayingConditions{}, ErrInvalidReviewQuota
	}
	if !tier.Valid() {
		return PlayingConditions{}, ErrInvalidCameraTier
	}
	for _, dt := range decisionTypes {
		if !dt.Valid() {
			return PlayingConditions{}, ErrInvalidDecisionType
		}
	}
	types := make([]DecisionType, len(decisionTypes))
	copy(types, decisionTypes)
	return PlayingConditions{
		ReviewQuotaPerInnings: reviewQuotaPerInnings,
		DecisionTypesEnabled:  types,
		CameraTier:            tier,
	}, nil
}
