package portfolio

import (
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
)

func PositionFromBond(
	b bonds.BondRecord,
	lots int,
	purchaseDate time.Time,
	source PositionSourceType,
) PortfolioPosition {
	cleanPct := 0.0
	if b.LastPrice != nil {
		cleanPct = *b.LastPrice
	}
	dirtyPerBond := 0.0
	if d := b.DirtyPriceRub(); d != nil {
		dirtyPerBond = *d
	}
	aci := 0.0
	if b.AccruedInterest != nil {
		aci = *b.AccruedInterest
	}
	bondsCount := lots * b.LotSize
	var offerDate *time.Time
	if b.OfferDate != nil && !b.OfferDate.Before(purchaseDate) {
		od := *b.OfferDate
		offerDate = &od
		if PutOfferBuyBlocked(b, purchaseDate) != nil {
			offerDate = nil
		}
	}
	pos := PortfolioPosition{
		ISIN:                  b.ISIN,
		Secid:                 b.Secid,
		Name:                  b.Name,
		Lots:                  lots,
		LotSize:               b.LotSize,
		PurchaseCleanPricePct: cleanPct,
		PurchaseDirtyPriceRub: dirtyPerBond,
		PurchaseACIRub:        aci,
		PurchaseDate:          purchaseDate,
		PurchaseAmountRub:     dirtyPerBond * float64(bondsCount),
		CouponRate:            b.CouponRate,
		CouponValue:           b.CouponValue,
		FaceValue:             b.FaceValue,
		MaturityDate:          b.MaturityDate,
		OfferDate:             offerDate,
		CouponPeriodDays:      b.CouponPeriodDays,
		NextCouponDate:        b.NextCouponDate,
		Source:                source,
		PutOfferDecision:      bonds.PutOfferPending,
	}
	if offerDate != nil {
		pos.OfferSubmissionStart = b.OfferSubmissionStart
		pos.OfferSubmissionEnd = b.OfferSubmissionEnd
		pos.OfferPricePct = b.OfferPricePct
	}
	if b.FIGI != "" {
		pos.FIGI = &b.FIGI
	}
	return pos
}

func SyncPutOfferFromBond(position *PortfolioPosition, b bonds.BondRecord) {
	if b.OfferDate == nil || b.OfferDate.Before(position.PurchaseDate) {
		return
	}
	prev := position.OfferDate
	position.OfferDate = b.OfferDate
	position.OfferSubmissionStart = b.OfferSubmissionStart
	position.OfferSubmissionEnd = b.OfferSubmissionEnd
	position.OfferPricePct = b.OfferPricePct
	if prev == nil || !prev.Equal(*b.OfferDate) {
		position.PutOfferDecision = bonds.PutOfferPending
	}
}

// SyncCouponFromBond refreshes coupon payment fields from the live bond record.
func SyncCouponFromBond(position *PortfolioPosition, b bonds.BondRecord) {
	if b.CouponValue != nil && *b.CouponValue > 0 {
		v := *b.CouponValue
		position.CouponValue = &v
	}
	if b.CouponRate != nil && *b.CouponRate > 0 {
		r := *b.CouponRate
		position.CouponRate = &r
	}
	if b.CouponPeriodDays != nil && *b.CouponPeriodDays > 0 {
		p := *b.CouponPeriodDays
		position.CouponPeriodDays = &p
	}
	if b.NextCouponDate != nil {
		d := *b.NextCouponDate
		position.NextCouponDate = &d
	}
}

func PositionEndDate(
	position PortfolioPosition,
	horizon time.Time,
	today time.Time,
	assumeBestPutOutcome bool,
) *time.Time {
	if position.OfferDate != nil &&
		!position.OfferDate.After(horizon) &&
		PositionPlansPutExit(position, today, assumeBestPutOutcome) {
		return position.OfferDate
	}
	return position.MaturityDate
}

func OpenPositions(positions []PortfolioPosition) []PortfolioPosition {
	return append([]PortfolioPosition(nil), positions...)
}
