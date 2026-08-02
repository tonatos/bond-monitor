package market_signals

import "sort"

// SpreadWideningPolicy thresholds for temporal credit-spread widening vs peers.
type SpreadWideningPolicy struct {
	MinPeers           int
	MinExcessSpreadPP  float64
}

// DefaultSpreadWideningPolicy requires a peer baseline and ≥5 п.п. excess widening.
var DefaultSpreadWideningPolicy = SpreadWideningPolicy{
	MinPeers:          5,
	MinExcessSpreadPP: 5.0,
}

// Median returns the median of values, or ok=false when the slice is empty.
func Median(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	if len(cp)%2 == 1 {
		return cp[len(cp)/2], true
	}
	return (cp[len(cp)/2-1] + cp[len(cp)/2]) / 2, true
}

// IsSpreadWidening reports whether bond spread widened vs peer median by policy.
// Empty / insufficient peer samples never alert (median([]) must not be treated as 0).
func IsSpreadWidening(bondSpreadChangePP float64, peerSpreadChanges []float64, policy SpreadWideningPolicy) (excessPP float64, peerMedian float64, ok bool) {
	minPeers := policy.MinPeers
	if minPeers <= 0 {
		minPeers = DefaultSpreadWideningPolicy.MinPeers
	}
	minExcess := policy.MinExcessSpreadPP
	if minExcess <= 0 {
		minExcess = DefaultSpreadWideningPolicy.MinExcessSpreadPP
	}
	if len(peerSpreadChanges) < minPeers {
		return 0, 0, false
	}
	peerMedian, ok = Median(peerSpreadChanges)
	if !ok {
		return 0, 0, false
	}
	excessPP = bondSpreadChangePP - peerMedian
	if excessPP < minExcess {
		return excessPP, peerMedian, false
	}
	return excessPP, peerMedian, true
}
