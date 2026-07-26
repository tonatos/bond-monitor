package trading

import (
	"math"
	"sort"
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/portfolio"
	"github.com/tonatos/instrumenta/backend/internal/domain/shared"
)

const minAnnualizeDays = 7

// ActualPerformance summarizes fact portfolio returns for the UI «Результат» card.
type ActualPerformance struct {
	TotalValueRub      shared.Rub
	NetProfitRub       shared.Rub
	FundedRub          shared.Rub
	AnnualYieldPct     *float64
	XIRRPct            *float64 // wire-compat: same as AnnualYieldPct
	CouponsReceivedRub shared.Rub
	TaxPaidRub         shared.Rub
	CommissionPaidRub  shared.Rub
	RealizedProfitRub  shared.Rub // wire-compat: same as NetProfitRub
	UnrealizedValueRub shared.Rub // holdings MTM only (without cash)
	InvestedRub        shared.Rub // wire-compat: same as FundedRub
	ReceivedRub        shared.Rub
	AsOf               string
}

func operationKind(opType string) string {
	return operationCashflowKinds[opType]
}

func isFundingOperationType(opType string) bool {
	k := operationKind(opType)
	return k == "deposit" || k == "withdrawal"
}

func isPurchaseOperationType(opType string) bool {
	return operationKind(opType) == "purchase"
}

func isSaleOperationType(opType string) bool {
	return operationKind(opType) == "sale"
}

func isExecuted(op BrokerOperation) bool {
	return op.State == "" || op.State == "OPERATION_STATE_EXECUTED"
}

// EpisodeStart is T0 for trading performance: TradingStartedAt, else CreatedAt.
func EpisodeStart(p portfolio.Portfolio) time.Time {
	if p.TradingStartedAt != nil && *p.TradingStartedAt != "" {
		if t, err := time.Parse(time.RFC3339, *p.TradingStartedAt); err == nil {
			return t.UTC()
		}
	}
	if p.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, p.CreatedAt); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func portfolioFigiSet(p portfolio.Portfolio) map[string]bool {
	figis := make(map[string]bool)
	for _, pos := range p.Positions {
		if pos.FIGI != nil && *pos.FIGI != "" {
			figis[*pos.FIGI] = true
		}
	}
	return figis
}

func estimateCurrentValue(p portfolio.Portfolio, snapshot BrokerSnapshot) shared.Rub {
	total := 0.0
	portfolioFigis := portfolioFigiSet(p)
	for figi, brokerPos := range snapshot.BondPositions {
		if !portfolioFigis[figi] {
			continue
		}
		var position *portfolio.PortfolioPosition
		for i := range p.Positions {
			if p.Positions[i].FIGI != nil && *p.Positions[i].FIGI == figi {
				position = &p.Positions[i]
				break
			}
		}
		if position == nil {
			continue
		}
		var cleanPerBond float64
		if brokerPos.CurrentPricePct == nil {
			cleanPerBond = position.FaceValue
		} else {
			cleanPerBond = float64(*brokerPos.CurrentPricePct) / 100 * position.FaceValue
		}
		nkd := 0.0
		if brokerPos.CurrentNKDRub != nil {
			nkd = float64(*brokerPos.CurrentNKDRub)
		}
		total += (cleanPerBond + nkd) * float64(brokerPos.Quantity)
	}
	return shared.Rub(total)
}

func fullPortfolioValue(p portfolio.Portfolio, snapshot BrokerSnapshot) shared.Rub {
	return shared.Rub(float64(estimateCurrentValue(p, snapshot)) + float64(snapshot.MoneyRub))
}

func currentQtyByFigi(p portfolio.Portfolio, snapshot BrokerSnapshot) map[string]int {
	figis := portfolioFigiSet(p)
	qty := make(map[string]int)
	for figi, pos := range snapshot.BondPositions {
		if !figis[figi] {
			continue
		}
		qty[figi] = pos.Quantity
	}
	return qty
}

// reverseStateToT0 walks ops after T0 newest→oldest and undoes cash/qty effects.
func reverseStateToT0(
	cash float64,
	qty map[string]int,
	ops []BrokerOperation,
	t0 time.Time,
	figis map[string]bool,
) (float64, map[string]int) {
	var after []BrokerOperation
	for _, op := range ops {
		if !isExecuted(op) || !op.Date.After(t0) {
			continue
		}
		after = append(after, op)
	}
	sort.Slice(after, func(i, j int) bool {
		if after[i].Date.Equal(after[j].Date) {
			return after[i].ID > after[j].ID
		}
		return after[i].Date.After(after[j].Date)
	})
	for _, op := range after {
		if op.PaymentRub != nil {
			cash -= float64(*op.PaymentRub)
		}
		if op.FIGI == "" || (len(figis) > 0 && !figis[op.FIGI]) {
			continue
		}
		if isPurchaseOperationType(op.Type) {
			qty[op.FIGI] -= op.Quantity
		} else if isSaleOperationType(op.Type) {
			qty[op.FIGI] += op.Quantity
		}
	}
	return cash, qty
}

// avgCostPerUnit builds average cost for each FIGI from executed buys/sells with Date ≤ T0.
func avgCostPerUnit(ops []BrokerOperation, t0 time.Time, figis map[string]bool) map[string]float64 {
	type lotState struct {
		qty  int
		cost float64
	}
	state := make(map[string]*lotState)
	var before []BrokerOperation
	for _, op := range ops {
		if !isExecuted(op) || op.Date.After(t0) {
			continue
		}
		if op.FIGI == "" || (len(figis) > 0 && !figis[op.FIGI]) {
			continue
		}
		if !isPurchaseOperationType(op.Type) && !isSaleOperationType(op.Type) {
			continue
		}
		before = append(before, op)
	}
	sort.Slice(before, func(i, j int) bool {
		if before[i].Date.Equal(before[j].Date) {
			return before[i].ID < before[j].ID
		}
		return before[i].Date.Before(before[j].Date)
	})
	for _, op := range before {
		st := state[op.FIGI]
		if st == nil {
			st = &lotState{}
			state[op.FIGI] = st
		}
		if isPurchaseOperationType(op.Type) {
			if op.PaymentRub == nil || op.Quantity <= 0 {
				continue
			}
			st.cost += -float64(*op.PaymentRub)
			st.qty += op.Quantity
			continue
		}
		// sale
		if op.Quantity <= 0 || st.qty <= 0 {
			continue
		}
		sellQty := op.Quantity
		if sellQty > st.qty {
			sellQty = st.qty
		}
		avg := st.cost / float64(st.qty)
		st.cost -= avg * float64(sellQty)
		st.qty -= sellQty
	}
	out := make(map[string]float64)
	for figi, st := range state {
		if st.qty > 0 {
			out[figi] = st.cost / float64(st.qty)
		}
	}
	return out
}

const fundingWaveMaxGapDays = 14

func fundingAfterT0(ops []BrokerOperation, t0 time.Time) float64 {
	total := 0.0
	for _, op := range ops {
		if !isExecuted(op) || !op.Date.After(t0) {
			continue
		}
		if op.PaymentRub == nil || !isFundingOperationType(op.Type) {
			continue
		}
		total += float64(*op.PaymentRub)
	}
	return total
}

// fundingWaveBeforeT0 sums contiguous external funding ending at T0 (gap ≤ fundingWaveMaxGapDays).
// Covers «пополнил, потом привязал портфель»: deposits on/before T0 that cost-basis opening may understate.
func fundingWaveBeforeT0(ops []BrokerOperation, t0 time.Time) float64 {
	if t0.IsZero() {
		return 0
	}
	type funded struct {
		date time.Time
		pay  float64
	}
	var items []funded
	for _, op := range ops {
		if !isExecuted(op) || op.Date.After(t0) {
			continue
		}
		if op.PaymentRub == nil || !isFundingOperationType(op.Type) {
			continue
		}
		items = append(items, funded{date: op.Date, pay: float64(*op.PaymentRub)})
	}
	if len(items) == 0 {
		return 0
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].date.Equal(items[j].date) {
			return items[i].pay > items[j].pay
		}
		return items[i].date.After(items[j].date)
	})
	total := 0.0
	cursor := t0
	started := false
	for _, it := range items {
		if !started {
			if shared.DaysBetween(it.date, t0) > fundingWaveMaxGapDays {
				continue
			}
			started = true
			total += it.pay
			cursor = it.date
			continue
		}
		if shared.DaysBetween(it.date, cursor) > fundingWaveMaxGapDays {
			break
		}
		total += it.pay
		cursor = it.date
	}
	return total
}

// EpisodeCapital is the single owner of trading «капитал».
// base = max(opening equity at T0, contiguous funding wave ≤ T0) + net funding after T0.
// Used by plan.invested_capital_rub (header / forecast «вложено») and performance.funded_rub.
func EpisodeCapital(p portfolio.Portfolio, snapshot BrokerSnapshot, operations []BrokerOperation) shared.Rub {
	t0 := EpisodeStart(p)
	figis := portfolioFigiSet(p)
	cash := float64(snapshot.MoneyRub)
	qty := currentQtyByFigi(p, snapshot)
	if !t0.IsZero() {
		cash, qty = reverseStateToT0(cash, qty, operations, t0, figis)
	}
	avg := avgCostPerUnit(operations, t0, figis)
	opening := cash
	for figi, q := range qty {
		if q <= 0 {
			continue
		}
		opening += float64(q) * avg[figi]
	}
	base := opening
	delta := 0.0
	if !t0.IsZero() {
		if wave := fundingWaveBeforeT0(operations, t0); wave > base {
			base = wave
		}
		delta = fundingAfterT0(operations, t0)
	}
	return shared.Rub(base + delta)
}

func annualizeROI(profit, funded shared.Rub, daysElapsed int) *float64 {
	if funded <= 0 || daysElapsed < minAnnualizeDays {
		return nil
	}
	pct := float64(profit) / float64(funded) * (365.0 / float64(daysElapsed)) * 100.0
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return nil
	}
	return &pct
}

func sumPaymentsByType(ops []BrokerOperation, types map[string]bool, figis map[string]bool) shared.Rub {
	total := 0.0
	for _, op := range ops {
		if !types[op.Type] {
			continue
		}
		if !isExecuted(op) {
			continue
		}
		if len(figis) > 0 && op.FIGI != "" && !figis[op.FIGI] {
			continue
		}
		if op.PaymentRub == nil {
			continue
		}
		total += float64(*op.PaymentRub)
	}
	return shared.Rub(total)
}

// SummarizeActualPerformance builds the UI performance card (fact metrics).
func SummarizeActualPerformance(
	p portfolio.Portfolio,
	snapshot BrokerSnapshot,
	operations []BrokerOperation,
	asOf time.Time,
) ActualPerformance {
	asOf = shared.DateOnly(asOf)

	holdingsMTM := estimateCurrentValue(p, snapshot)
	totalValue := fullPortfolioValue(p, snapshot)
	funded := EpisodeCapital(p, snapshot, operations)
	netProfit := shared.Rub(float64(totalValue) - float64(funded))

	var annual *float64
	t0 := EpisodeStart(p)
	if !t0.IsZero() {
		daysElapsed := shared.DaysBetween(t0, asOf)
		annual = annualizeROI(netProfit, funded, daysElapsed)
	}

	figis := portfolioFigiSet(p)
	coupons := sumPaymentsByType(operations, map[string]bool{"OPERATION_TYPE_COUPON": true}, figis)
	taxTypes := map[string]bool{
		"OPERATION_TYPE_BOND_TAX": true, "OPERATION_TYPE_BOND_TAX_PROGRESSIVE": true,
		"OPERATION_TYPE_TAX": true, "OPERATION_TYPE_TAX_PROGRESSIVE": true,
		"OPERATION_TYPE_TAX_CORRECTION": true, "OPERATION_TYPE_TAX_CORRECTION_COUPON": true,
	}
	taxPaid := shared.Rub(-float64(sumPaymentsByType(operations, taxTypes, figis)))
	commissionTypes := map[string]bool{
		"OPERATION_TYPE_BROKER_FEE": true, "OPERATION_TYPE_SERVICE_FEE": true, "OPERATION_TYPE_OTHER_FEE": true,
	}
	commission := shared.Rub(-float64(sumPaymentsByType(operations, commissionTypes, figis)))
	inflowTypes := map[string]bool{
		"OPERATION_TYPE_SELL": true, "OPERATION_TYPE_COUPON": true,
		"OPERATION_TYPE_BOND_REPAYMENT": true, "OPERATION_TYPE_BOND_REPAYMENT_FULL": true,
	}
	received := sumPaymentsByType(operations, inflowTypes, figis)

	return ActualPerformance{
		TotalValueRub:      totalValue,
		NetProfitRub:       netProfit,
		FundedRub:          funded,
		AnnualYieldPct:     annual,
		XIRRPct:            annual,
		CouponsReceivedRub: coupons,
		TaxPaidRub:         taxPaid,
		CommissionPaidRub:  commission,
		RealizedProfitRub:  netProfit,
		UnrealizedValueRub: holdingsMTM,
		InvestedRub:        funded,
		ReceivedRub:        received,
		AsOf:               asOf.UTC().Format(time.RFC3339),
	}
}
