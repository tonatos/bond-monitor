package portfolio_test

import (
	"testing"

	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
	"github.com/tonatos/instrumenta/backend/internal/domain/portfolio"
	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
)

func TestCouponPaymentPerEventUsesCouponValue(t *testing.T) {
	period := 30
	value := 15.35
	pos := portfolio.PortfolioPosition{
		Lots: 99, LotSize: 1, FaceValue: 1000,
		CouponPeriodDays: &period, CouponValue: &value,
	}
	got := portfolio.CouponPaymentPerEvent(pos)
	want := 15.35 * 99
	if got < want-0.01 || got > want+0.01 {
		t.Fatalf("payment = %v, want %v", got, want)
	}
}

func TestCouponDatesInRangeWithCouponValueWithoutRate(t *testing.T) {
	period := 30
	value := 15.35
	next := shared.MustParseDate("2026-08-06")
	pos := portfolio.PortfolioPosition{
		PurchaseDate:     shared.MustParseDate("2026-07-26"),
		CouponPeriodDays: &period,
		CouponValue:      &value,
		NextCouponDate:   &next,
		Lots:             1, LotSize: 1, FaceValue: 1000,
	}
	dates := portfolio.CouponDatesInRange(pos, shared.MustParseDate("2026-08-06"))
	if len(dates) != 1 || !dates[0].Equal(next) {
		t.Fatalf("dates = %v, want [%v]", dates, next)
	}
}

func TestSimulationSyncsCouponValueFromUniverse(t *testing.T) {
	today := shared.MustParseDate("2026-07-26")
	maturity := shared.MustParseDate("2026-08-06")
	horizon := shared.MustParseDate("2026-08-06")
	period := 30
	next := maturity
	couponValue := 15.35

	pos := portfolio.PortfolioPosition{
		ISIN: "RU000A109908", Secid: "RU000A109908", Name: "МВ ФИН 1Р5",
		Lots: 10, LotSize: 1, FaceValue: 1000,
		PurchaseDate: today, PurchaseAmountRub: 10040,
		PurchaseDirtyPriceRub: 1004, PurchaseCleanPricePct: 99.35,
		MaturityDate: &maturity, CouponPeriodDays: &period, NextCouponDate: &next,
		Source: portfolio.PositionSourceInitial,
		// no CouponRate / CouponValue on saved position
	}
	live := makeBond("RU000A109908", "МВ ФИН 1Р5", maturity, 99.35, 45, 50, bonds.BoolPtr(true), func(b *bonds.BondRecord) {
		b.CouponPeriodDays = &period
		b.NextCouponDate = &next
		b.CouponValue = &couponValue
		b.AccruedInterest = bonds.FloatPtr(10.64)
	})
	p := portfolio.Portfolio{
		InitialAmountRub: 10040, HorizonDate: horizon, CashBalanceRub: 0,
		Mode: portfolio.PortfolioModeSimulation, RiskProfile: portfolio.RiskProfileNormal,
		Positions: []portfolio.PortfolioPosition{pos},
	}
	plan := portfolio.BuildPlan(
		p, []bonds.BondRecord{live}, today, 16, 0.13,
		portfolio.NewSimulationPlanContext(p, true),
		portfolio.DefaultDurationPolicy,
	)
	var couponTotal float64
	for _, ev := range plan.Events {
		if ev.Kind == "coupon" {
			couponTotal += ev.AmountRub
		}
	}
	wantNet := 15.35 * 10 * (1 - 0.13)
	if couponTotal < wantNet-0.01 || couponTotal > wantNet+0.01 {
		t.Fatalf("coupon total = %v, want ~%v; events=%+v", couponTotal, wantNet, plan.Events)
	}
}

func TestCalculateBondHoldUsesCouponValue(t *testing.T) {
	today := shared.MustParseDate("2026-07-26")
	maturity := shared.MustParseDate("2026-08-06")
	period := 30
	b := makeBond("RU000A109908", "МВ ФИН 1Р5", maturity, 99.35, 45, 50, bonds.BoolPtr(true), func(br *bonds.BondRecord) {
		br.CouponPeriodDays = &period
		br.NextCouponDate = &maturity
		br.CouponValue = bonds.FloatPtr(15.35)
		br.AccruedInterest = bonds.FloatPtr(10.64)
		br.EffectiveDate = &maturity
	})
	hold := portfolio.CalculateBondHold(b, 99, today)
	if hold == nil {
		t.Fatal("expected hold result")
	}
	wantCoupons := 15.35 * 99
	if hold.CouponIncomeRub < wantCoupons-0.01 || hold.CouponIncomeRub > wantCoupons+0.01 {
		t.Fatalf("coupons = %v, want %v", hold.CouponIncomeRub, wantCoupons)
	}
	if hold.ProfitRub <= 0 {
		t.Fatalf("expected positive profit with coupons, got %v", hold.ProfitRub)
	}
}
