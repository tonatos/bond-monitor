package bonds_test

import (
	"testing"
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
)

func TestResolveCouponValueFromSchedulePrefersNearestFuture(t *testing.T) {
	today := shared.MustParseDate("2026-07-26")
	past := 12.0
	near := 15.35
	far := 16.0
	payments := []bonds.CouponPayment{
		{PaymentDate: ptrDate(shared.MustParseDate("2026-06-06")), AmountRub: &past},
		{PaymentDate: ptrDate(shared.MustParseDate("2026-09-06")), AmountRub: &far},
		{PaymentDate: ptrDate(shared.MustParseDate("2026-08-06")), AmountRub: &near},
		{PaymentDate: ptrDate(shared.MustParseDate("2026-10-06")), AmountRub: nil},
	}
	got := bonds.ResolveCouponValueFromSchedule(payments, today)
	if got == nil || *got != near {
		t.Fatalf("expected nearest future %.2f, got %v", near, got)
	}
}

func TestResolveCouponValueFromScheduleFallsBackToLatestPast(t *testing.T) {
	today := shared.MustParseDate("2026-07-26")
	older := 10.0
	newer := 15.35
	payments := []bonds.CouponPayment{
		{PaymentDate: ptrDate(shared.MustParseDate("2026-05-06")), AmountRub: &older},
		{PaymentDate: ptrDate(shared.MustParseDate("2026-06-06")), AmountRub: &newer},
		{PaymentDate: ptrDate(shared.MustParseDate("2026-08-06")), AmountRub: nil},
	}
	got := bonds.ResolveCouponValueFromSchedule(payments, today)
	if got == nil || *got != newer {
		t.Fatalf("expected latest past %.2f, got %v", newer, got)
	}
}

func TestResolveCouponValueFromScheduleAllNull(t *testing.T) {
	today := shared.MustParseDate("2026-07-26")
	payments := []bonds.CouponPayment{
		{PaymentDate: ptrDate(shared.MustParseDate("2026-08-06")), AmountRub: nil},
		{PaymentDate: ptrDate(shared.MustParseDate("2026-06-06")), AmountRub: bonds.FloatPtr(0)},
	}
	if got := bonds.ResolveCouponValueFromSchedule(payments, today); got != nil {
		t.Fatalf("expected nil, got %v", *got)
	}
}

func TestNeedsCouponValueEnrichment(t *testing.T) {
	if bonds.NeedsCouponValueEnrichment(bonds.BondRecord{FIGI: "F"}) != true {
		t.Fatal("expected enrichment when figi set and no value/rate")
	}
	if bonds.NeedsCouponValueEnrichment(bonds.BondRecord{
		FIGI: "F", CouponRate: bonds.FloatPtr(20),
	}) {
		t.Fatal("rate present — skip enrichment")
	}
	if bonds.NeedsCouponValueEnrichment(bonds.BondRecord{
		FIGI: "F", CouponValue: bonds.FloatPtr(15),
	}) {
		t.Fatal("value present — skip enrichment")
	}
	if bonds.NeedsCouponValueEnrichment(bonds.BondRecord{}) {
		t.Fatal("no figi — skip enrichment")
	}
}

func ptrDate(t time.Time) *time.Time { return &t }
