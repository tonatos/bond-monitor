package bonds

import (
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
)

// NeedsCouponValueEnrichment reports whether CouponValue should be filled from
// an external coupon schedule (T-Invest). Skip when MOEX already has a usable
// value or rate for payment math.
func NeedsCouponValueEnrichment(b BondRecord) bool {
	if b.FIGI == "" {
		return false
	}
	if b.CouponValue != nil && *b.CouponValue > 0 {
		return false
	}
	if b.CouponRate != nil && *b.CouponRate > 0 {
		return false
	}
	return true
}

// ResolveCouponValueFromSchedule picks a per-bond coupon amount from schedule:
// nearest future PayOneBond > 0, else latest past PayOneBond > 0.
func ResolveCouponValueFromSchedule(payments []CouponPayment, today time.Time) *float64 {
	today = shared.DateOnly(today)
	var (
		bestFuture     *float64
		bestFutureDate time.Time
		haveFuture     bool
		bestPast       *float64
		bestPastDate   time.Time
		havePast       bool
	)
	for _, p := range payments {
		if p.AmountRub == nil || *p.AmountRub <= 0 || p.PaymentDate == nil {
			continue
		}
		d := shared.DateOnly(*p.PaymentDate)
		amt := *p.AmountRub
		if !d.Before(today) {
			if !haveFuture || d.Before(bestFutureDate) {
				v := amt
				bestFuture = &v
				bestFutureDate = d
				haveFuture = true
			}
			continue
		}
		if !havePast || d.After(bestPastDate) {
			v := amt
			bestPast = &v
			bestPastDate = d
			havePast = true
		}
	}
	if haveFuture {
		return bestFuture
	}
	if havePast {
		return bestPast
	}
	return nil
}

// ApplyCouponValue sets CouponValue when the resolved amount is usable.
func ApplyCouponValue(bond *BondRecord, value *float64) {
	if bond == nil || value == nil || *value <= 0 {
		return
	}
	if bond.CouponValue != nil && *bond.CouponValue > 0 {
		return
	}
	v := *value
	bond.CouponValue = &v
}
