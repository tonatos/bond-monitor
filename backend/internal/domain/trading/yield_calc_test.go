package trading_test

import (
	"math"
	"testing"
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/portfolio"
	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
	"github.com/tonatos/instrumenta/backend/internal/domain/trading"
	"github.com/tonatos/instrumenta/backend/internal/domain/trading/testutil"
)

func rubPtr(v float64) *shared.Rub {
	r := shared.Rub(v)
	return &r
}

func pricePct(v float64) *shared.PriceUnitPct {
	p := shared.PriceUnitPct(v)
	return &p
}

func strPtr(s string) *string { return &s }

// aa19: empty-ish account at attach morning, then +50k; capital must be exactly 250k.
func TestSummarizeActualPerformanceAA19EpisodeCapital(t *testing.T) {
	asOf := shared.MustParseDate("2026-07-25")
	t0 := "2026-07-07T10:21:47Z"
	figi := "FIGI-A"
	p := testutil.MakePortfolio(func(p *portfolio.Portfolio) {
		p.CreatedAt = t0
		p.TradingStartedAt = strPtr(t0)
		p.Mode = portfolio.PortfolioModeTrading
		p.Positions = []portfolio.PortfolioPosition{{
			ISIN: "RU000A1", Name: "Bond", Lots: 250, LotSize: 1,
			FaceValue: 1000, FIGI: &figi,
		}}
	})
	// value_now = 250_000 MTM + 3_825 cash = 253_825
	snap := testutil.MakeAccountSnapshot(3_825, func(s *trading.BrokerSnapshot) {
		pos := testutil.BondPosition(figi, 250, 250)
		pos.CurrentPricePct = pricePct(100)
		nkd := shared.Rub(0)
		pos.CurrentNKDRub = &nkd
		s.BondPositions[figi] = pos
	})
	ops := []trading.BrokerOperation{
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-07"), PaymentRub: rubPtr(180_000),
		},
		{
			Type: "OPERATION_TYPE_INP_MULTI", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-07"), PaymentRub: rubPtr(20_000),
		},
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-08"), PaymentRub: rubPtr(50_000),
		},
		{
			Type: "OPERATION_TYPE_BUY", State: "OPERATION_STATE_EXECUTED", FIGI: figi,
			Date: shared.MustParseDate("2026-07-09"), Quantity: 250, PaymentRub: rubPtr(-250_000),
		},
		{
			Type: "OPERATION_TYPE_COUPON", State: "OPERATION_STATE_EXECUTED", FIGI: figi,
			Date: shared.MustParseDate("2026-07-20"), PaymentRub: rubPtr(3_825),
		},
		// Lifetime noise — must not enter episode capital.
		{
			Type: "OPERATION_TYPE_OUTPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2023-01-15"), PaymentRub: rubPtr(-380_000),
		},
	}

	perf := trading.SummarizeActualPerformance(p, snap, ops, asOf)
	if math.Abs(float64(perf.TotalValueRub)-253_825) > 0.01 {
		t.Fatalf("total value: got %.2f want 253825", perf.TotalValueRub)
	}
	if math.Abs(float64(perf.FundedRub)-250_000) > 0.01 {
		t.Fatalf("capital: got %.2f want 250000", perf.FundedRub)
	}
	if math.Abs(float64(perf.NetProfitRub)-3_825) > 0.01 {
		t.Fatalf("profit: got %.2f want 3825", perf.NetProfitRub)
	}
	if perf.AnnualYieldPct == nil {
		t.Fatal("expected annual yield")
	}
	wantAnnual := 3825.0 / 250_000 * 365.0 / 18.0 * 100.0
	if math.Abs(*perf.AnnualYieldPct-wantAnnual) > 0.05 {
		t.Fatalf("annual: got %.4f want %.4f (~31%%)", *perf.AnnualYieldPct, wantAnnual)
	}
	if *perf.AnnualYieldPct < 25 || *perf.AnnualYieldPct > 35 {
		t.Fatalf("annual out of ~31%% band: %.2f", *perf.AnnualYieldPct)
	}
}

// 2cd: attach to existing holdings; lifetime funding ignored; capital = opening at T0.
func TestSummarizeActualPerformanceAttachExistingHoldings(t *testing.T) {
	asOf := shared.MustParseDate("2026-07-25")
	t0 := "2026-07-08T08:29:34Z"
	figi := "FIGI-B"
	p := testutil.MakePortfolio(func(p *portfolio.Portfolio) {
		p.CreatedAt = t0
		p.TradingStartedAt = strPtr(t0)
		p.Mode = portfolio.PortfolioModeTrading
		p.Positions = []portfolio.PortfolioPosition{{
			ISIN: "RU000A2", Name: "Bond", Lots: 408, LotSize: 1,
			FaceValue: 1000, FIGI: &figi,
		}}
	})
	// value_now = 412_256 MTM + 4_087 cash = 416_343
	snap := testutil.MakeAccountSnapshot(4_087, func(s *trading.BrokerSnapshot) {
		pos := testutil.BondPosition(figi, 408, 408)
		// 412256 / 408 / 1000 * 100 = 101.043…%
		pos.CurrentPricePct = pricePct(412_256.0 / 408.0 / 10.0)
		nkd := shared.Rub(0)
		pos.CurrentNKDRub = &nkd
		s.BondPositions[figi] = pos
	})
	ops := []trading.BrokerOperation{
		// Lifetime funding noise (would produce ~36k profit under old model).
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2025-11-01"), PaymentRub: rubPtr(282_774),
		},
		{
			Type: "OPERATION_TYPE_OUTPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2025-11-17"), PaymentRub: rubPtr(-207_827.64),
		},
		{
			Type: "OPERATION_TYPE_BUY", State: "OPERATION_STATE_EXECUTED", FIGI: figi,
			Date: shared.MustParseDate("2026-06-01"), Quantity: 408, PaymentRub: rubPtr(-408_000),
		},
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-01"), PaymentRub: rubPtr(4_087),
		},
	}

	perf := trading.SummarizeActualPerformance(p, snap, ops, asOf)
	wantCapital := 408_000.0 + 4_087.0
	if math.Abs(float64(perf.FundedRub)-wantCapital) > 0.05 {
		t.Fatalf("capital: got %.2f want %.2f (opening, not lifetime funding)", perf.FundedRub, wantCapital)
	}
	wantProfit := float64(perf.TotalValueRub) - wantCapital
	if math.Abs(float64(perf.NetProfitRub)-wantProfit) > 0.05 {
		t.Fatalf("profit: got %.2f want %.2f", perf.NetProfitRub, wantProfit)
	}
	if wantProfit > 20_000 {
		t.Fatalf("profit too large for attach episode (lifetime regression): %.2f", wantProfit)
	}
	if math.Abs(wantProfit-4_256) > 1 {
		t.Fatalf("expected ~4256 MTM profit, got %.2f", wantProfit)
	}
}

func TestSummarizeActualPerformanceWithdrawalAfterT0ReducesCapital(t *testing.T) {
	asOf := shared.MustParseDate("2026-07-25")
	t0 := "2026-07-01T00:00:00Z"
	p := testutil.MakePortfolio(func(p *portfolio.Portfolio) {
		p.CreatedAt = t0
		p.TradingStartedAt = strPtr(t0)
		p.Mode = portfolio.PortfolioModeTrading
	})
	snap := testutil.MakeAccountSnapshot(80_000)
	ops := []trading.BrokerOperation{
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC), PaymentRub: rubPtr(100_000),
		},
		{
			Type: "OPERATION_TYPE_OUT_MULTI", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-07"), PaymentRub: rubPtr(-20_000),
		},
	}

	perf := trading.SummarizeActualPerformance(p, snap, ops, asOf)
	if math.Abs(float64(perf.FundedRub)-80_000) > 0.01 {
		t.Fatalf("capital after withdrawal: got %.2f want 80000", perf.FundedRub)
	}
	if math.Abs(float64(perf.NetProfitRub)-0) > 0.01 {
		t.Fatalf("profit: got %.2f want 0", perf.NetProfitRub)
	}
}

func TestSummarizeActualPerformanceAnnualNilWhenTooYoung(t *testing.T) {
	asOf := shared.MustParseDate("2026-07-03")
	t0 := "2026-07-01T00:00:00Z"
	p := testutil.MakePortfolio(func(p *portfolio.Portfolio) {
		p.CreatedAt = t0
		p.TradingStartedAt = strPtr(t0)
		p.Mode = portfolio.PortfolioModeTrading
	})
	snap := testutil.MakeAccountSnapshot(101_000)
	ops := []trading.BrokerOperation{
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC), PaymentRub: rubPtr(100_000),
		},
		{
			Type: "OPERATION_TYPE_COUPON", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-02"), PaymentRub: rubPtr(1_000),
		},
	}

	perf := trading.SummarizeActualPerformance(p, snap, ops, asOf)
	if perf.AnnualYieldPct != nil {
		t.Fatalf("expected nil annual for <7 days from T0, got %v", *perf.AnnualYieldPct)
	}
	if math.Abs(float64(perf.FundedRub)-100_000) > 0.01 {
		t.Fatalf("capital: got %.2f want 100000", perf.FundedRub)
	}
	if math.Abs(float64(perf.NetProfitRub)-1_000) > 0.01 {
		t.Fatalf("profit: got %.2f want 1000", perf.NetProfitRub)
	}
}

func TestSummarizeActualPerformanceSkipsNonExecutedOps(t *testing.T) {
	asOf := shared.MustParseDate("2026-07-25")
	t0 := "2026-07-01T00:00:00Z"
	p := testutil.MakePortfolio(func(p *portfolio.Portfolio) {
		p.CreatedAt = t0
		p.TradingStartedAt = strPtr(t0)
		p.Mode = portfolio.PortfolioModeTrading
	})
	snap := testutil.MakeAccountSnapshot(10_000)
	ops := []trading.BrokerOperation{
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), PaymentRub: rubPtr(10_000),
		},
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_CANCELED",
			Date: shared.MustParseDate("2026-07-10"), PaymentRub: rubPtr(50_000),
		},
	}

	perf := trading.SummarizeActualPerformance(p, snap, ops, asOf)
	if math.Abs(float64(perf.FundedRub)-10_000) > 0.01 {
		t.Fatalf("canceled funding must not count: got %.2f", perf.FundedRub)
	}
}

func TestEpisodeCapitalMatchesPerformanceFunded(t *testing.T) {
	asOf := shared.MustParseDate("2026-07-25")
	t0 := "2026-07-07T10:21:47Z"
	figi := "FIGI-A"
	p := testutil.MakePortfolio(func(p *portfolio.Portfolio) {
		p.CreatedAt = t0
		p.TradingStartedAt = strPtr(t0)
		p.Mode = portfolio.PortfolioModeTrading
		p.Positions = []portfolio.PortfolioPosition{{
			ISIN: "RU000A1", Name: "Bond", Lots: 250, LotSize: 1,
			FaceValue: 1000, FIGI: &figi,
		}}
	})
	snap := testutil.MakeAccountSnapshot(3_825, func(s *trading.BrokerSnapshot) {
		pos := testutil.BondPosition(figi, 250, 250)
		pos.CurrentPricePct = pricePct(100)
		nkd := shared.Rub(0)
		pos.CurrentNKDRub = &nkd
		s.BondPositions[figi] = pos
	})
	ops := []trading.BrokerOperation{
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-07"), PaymentRub: rubPtr(180_000),
		},
		{
			Type: "OPERATION_TYPE_INP_MULTI", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-07"), PaymentRub: rubPtr(20_000),
		},
		{
			Type: "OPERATION_TYPE_INPUT", State: "OPERATION_STATE_EXECUTED",
			Date: shared.MustParseDate("2026-07-08"), PaymentRub: rubPtr(50_000),
		},
		{
			Type: "OPERATION_TYPE_BUY", State: "OPERATION_STATE_EXECUTED", FIGI: figi,
			Date: shared.MustParseDate("2026-07-09"), Quantity: 250, PaymentRub: rubPtr(-250_000),
		},
		{
			Type: "OPERATION_TYPE_COUPON", State: "OPERATION_STATE_EXECUTED", FIGI: figi,
			Date: shared.MustParseDate("2026-07-20"), PaymentRub: rubPtr(3_825),
		},
	}

	capital := trading.EpisodeCapital(p, snap, ops)
	perf := trading.SummarizeActualPerformance(p, snap, ops, asOf)
	if capital != perf.FundedRub {
		t.Fatalf("EpisodeCapital %.2f != performance.funded %.2f", capital, perf.FundedRub)
	}
	if math.Abs(float64(capital)-250_000) > 0.01 {
		t.Fatalf("shared capital: got %.2f want 250000", capital)
	}
}
