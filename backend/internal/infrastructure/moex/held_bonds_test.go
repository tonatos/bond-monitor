package moex

import (
	"testing"
	"time"
)

func TestBuildBondRecord_HeldModeIncludesMaturityToday(t *testing.T) {
	today := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	raw := map[string]any{
		"SECID": "RU000A100PB0", "SHORTNAME": "ЖКХРСЯ БО1", "FACEUNIT": "SUR",
		"MATDATE": "2026-07-28", "FACEVALUE": 1000.0, "LOTSIZE": 1.0, "LAST": 99.5, "YIELD": 20.0,
	}
	screener := buildBondRecord("RU000A100PB0", raw, today, nil, nil, bondBuildScreener)
	if screener != nil {
		t.Fatal("screener mode should drop bonds maturing today")
	}
	held := buildBondRecord("RU000A100PB0", raw, today, nil, nil, bondBuildHeld)
	if held == nil {
		t.Fatal("held mode should keep bonds maturing today")
	}
	if held.Secid != "RU000A100PB0" || held.ISIN != "RU000A100PB0" {
		t.Fatalf("bond = %+v", held)
	}
}

func TestBuildBondRecord_HeldModeIncludesPastMaturity(t *testing.T) {
	today := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	raw := map[string]any{
		"SECID": "RU000A100PB0", "SHORTNAME": "ЖКХРСЯ БО1", "FACEUNIT": "SUR",
		"MATDATE": "2026-07-28", "FACEVALUE": 1000.0, "LOTSIZE": 1.0, "LAST": 99.5,
	}
	held := buildBondRecord("RU000A100PB0", raw, today, nil, nil, bondBuildHeld)
	if held == nil {
		t.Fatal("held mode should keep recently matured bonds for open positions")
	}
	if held.DaysToMaturity == nil || *held.DaysToMaturity >= 0 {
		t.Fatalf("days = %v, want negative", held.DaysToMaturity)
	}
}
