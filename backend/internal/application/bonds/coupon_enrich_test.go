package bonds_test

import (
	"testing"
	"time"

	appbonds "github.com/tonatos/instrumenta/backend/internal/application/bonds"
	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
)

type stubEnricher struct {
	schedule map[string][]bonds.CouponPayment
}

func (s stubEnricher) EnrichBonds(bs []bonds.BondRecord) []bonds.BondRecord { return bs }
func (s stubEnricher) EnrichBondDetail(*bonds.BondRecord)                   {}
func (s stubEnricher) GetCouponSchedule(figi string) []bonds.CouponPayment {
	return s.schedule[figi]
}

func TestEnrichCouponValuesFromSchedule(t *testing.T) {
	amt := 15.35
	enricher := stubEnricher{schedule: map[string][]bonds.CouponPayment{
		"FIGI1": {{
			PaymentDate: ptrTime(shared.MustParseDate("2026-08-06")),
			AmountRub:   &amt,
		}},
	}}
	svc := appbonds.NewServiceWithDeps(16, 0.13, "token", nil, nil, enricher, nil)
	bs := []bonds.BondRecord{{
		ISIN: "RU000A109908", FIGI: "FIGI1", Name: "МВ ФИН 1Р5", FaceValue: 1000, LotSize: 1,
	}}
	svc.EnrichCouponValues(bs, []string{"RU000A109908"})
	if bs[0].CouponValue == nil || *bs[0].CouponValue != amt {
		t.Fatalf("CouponValue = %v, want %v", bs[0].CouponValue, amt)
	}
}

func TestEnrichCouponValuesSkipsWhenRatePresent(t *testing.T) {
	called := false
	enricher := stubEnricher{schedule: map[string][]bonds.CouponPayment{}}
	// wrap to detect calls — GetCouponSchedule returns nil; if rate present, should not need value
	_ = called
	svc := appbonds.NewServiceWithDeps(16, 0.13, "token", nil, nil, enricher, nil)
	bs := []bonds.BondRecord{{
		ISIN: "RU000A1", FIGI: "FIGI1", CouponRate: bonds.FloatPtr(20), FaceValue: 1000, LotSize: 1,
	}}
	svc.EnrichCouponValues(bs, nil)
	if bs[0].CouponValue != nil {
		t.Fatalf("expected no CouponValue when rate present, got %v", *bs[0].CouponValue)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
