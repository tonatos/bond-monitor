package market_signals_test

import (
	"testing"

	"github.com/tonatos/instrumenta/backend/internal/domain/market_signals"
)

func TestIsSpreadWidening_EmptyPeersNoAlert(t *testing.T) {
	_, _, ok := market_signals.IsSpreadWidening(8.0, nil, market_signals.DefaultSpreadWideningPolicy)
	if ok {
		t.Fatal("expected no alert when peer samples are empty")
	}
}

func TestIsSpreadWidening_TooFewPeersNoAlert(t *testing.T) {
	peers := []float64{0.1, 0.2, 0.0, -0.1} // 4 < MinPeers 5
	_, _, ok := market_signals.IsSpreadWidening(10.0, peers, market_signals.DefaultSpreadWideningPolicy)
	if ok {
		t.Fatal("expected no alert when peer sample < MinPeers")
	}
}

func TestIsSpreadWidening_ExcessOverThreshold(t *testing.T) {
	peers := []float64{0.0, 0.5, 1.0, 0.2, -0.3}
	excess, peerMedian, ok := market_signals.IsSpreadWidening(6.0, peers, market_signals.DefaultSpreadWideningPolicy)
	if !ok {
		t.Fatal("expected spread widening alert")
	}
	if peerMedian != 0.2 {
		t.Fatalf("unexpected peer median %.2f", peerMedian)
	}
	if excess < 5.0 {
		t.Fatalf("expected excess >= 5, got %.2f", excess)
	}
}

func TestIsSpreadWidening_BelowThreshold(t *testing.T) {
	peers := []float64{1.0, 1.0, 1.0, 1.0, 1.0}
	_, _, ok := market_signals.IsSpreadWidening(4.0, peers, market_signals.DefaultSpreadWideningPolicy)
	if ok {
		t.Fatal("expected no alert when excess < 5pp")
	}
}

func TestMedian_Empty(t *testing.T) {
	_, ok := market_signals.Median(nil)
	if ok {
		t.Fatal("expected ok=false for empty slice")
	}
}
