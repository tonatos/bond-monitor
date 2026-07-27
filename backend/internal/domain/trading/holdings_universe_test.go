package trading_test

import (
	"testing"
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
	"github.com/tonatos/instrumenta/backend/internal/domain/trading"
	"github.com/tonatos/instrumenta/backend/internal/domain/trading/testutil"
)

func TestBuildHoldingsMatchesBySecidWhenFIGIMissing(t *testing.T) {
	maturity := shared.MustParseDate("2026-07-28")
	snapshot := trading.BrokerSnapshot{
		BondPositions: map[string]trading.BrokerBondPosition{
			"BBGYAKUTV001": {
				FIGI: "BBGYAKUTV001", Ticker: "RU000A100PB0", Lots: 5, Quantity: 5,
				CurrentPricePct: pricePtr(99.5),
			},
		},
	}
	universe := []bonds.BondRecord{{
		ISIN: "RU000A100PB0", Secid: "RU000A100PB0", Name: "ЖКХРСЯ БО1",
		FaceValue: 1000, LotSize: 1, LastPrice: bonds.FloatPtr(99.5), MaturityDate: &maturity,
	}}
	holdings := trading.BuildHoldings(snapshot, universe)
	if len(holdings) != 1 {
		t.Fatalf("holdings = %+v", holdings)
	}
	if holdings[0].ISIN != "RU000A100PB0" {
		t.Fatalf("ISIN = %q", holdings[0].ISIN)
	}
}

func TestEffectiveTradingPositionsKeepsHeldBondWithoutFIGIInUniverse(t *testing.T) {
	today := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	maturity := shared.MustParseDate("2026-07-28")
	snapshot := trading.BrokerSnapshot{
		BondPositions: map[string]trading.BrokerBondPosition{
			"BBGYAKUTV001": {
				FIGI: "BBGYAKUTV001", Ticker: "RU000A100PB0", Lots: 5, Quantity: 5,
				AveragePricePct: pricePtr(99.5),
			},
		},
	}
	universe := []bonds.BondRecord{{
		ISIN: "RU000A100PB0", Secid: "RU000A100PB0", Name: "ЖКХРСЯ БО1",
		FaceValue: 1000, LotSize: 1, LastPrice: bonds.FloatPtr(99.5), MaturityDate: &maturity,
	}}
	p := testutil.MakePortfolio()
	positions := trading.EffectiveTradingPositions(p, snapshot, universe, today)
	if len(positions) != 1 {
		t.Fatalf("positions = %+v", positions)
	}
	if positions[0].ISIN != "RU000A100PB0" || positions[0].Lots != 5 {
		t.Fatalf("position = %+v", positions[0])
	}
}

func pricePtr(v float64) *shared.PriceUnitPct {
	p := shared.PriceUnitPct(v)
	return &p
}
