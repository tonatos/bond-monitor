package portfolio

import (
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
)

func hasCouponPaymentBasis(position PortfolioPosition) bool {
	if position.CouponValue != nil && *position.CouponValue > 0 {
		return true
	}
	return position.CouponRate != nil && *position.CouponRate > 0
}

// CouponDatesInRange returns coupon payment dates in (purchase_date, end_date].
func CouponDatesInRange(position PortfolioPosition, endDate time.Time) []time.Time {
	if position.CouponPeriodDays == nil || *position.CouponPeriodDays <= 0 {
		return nil
	}
	if !hasCouponPaymentBasis(position) {
		return nil
	}
	period := *position.CouponPeriodDays
	var current time.Time
	if position.NextCouponDate != nil {
		current = shared.DateOnly(*position.NextCouponDate)
		for !current.After(position.PurchaseDate) {
			current = shared.AddDays(current, period)
		}
	} else {
		current = shared.AddDays(position.PurchaseDate, period)
	}
	endDate = shared.DateOnly(endDate)
	var dates []time.Time
	for !current.After(endDate) {
		dates = append(dates, current)
		current = shared.AddDays(current, period)
	}
	return dates
}

// CouponPaymentPerEvent returns gross coupon payment per event in RUB.
func CouponPaymentPerEvent(position PortfolioPosition) float64 {
	if position.CouponValue != nil && *position.CouponValue > 0 {
		return *position.CouponValue * float64(position.BondsCount())
	}
	if position.CouponRate == nil || position.CouponPeriodDays == nil {
		return 0
	}
	perBond := position.FaceValue * (*position.CouponRate / 100) * (float64(*position.CouponPeriodDays) / 365)
	return perBond * float64(position.BondsCount())
}
