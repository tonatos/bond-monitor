package portfolio

import (
	"context"
	"fmt"
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/bonds"
	domain "github.com/tonatos/instrumenta/backend/internal/domain/portfolio"
	"github.com/tonatos/instrumenta/backend/internal/domain/trading"
	"github.com/tonatos/instrumenta/backend/internal/interfaces/auth"
)

// BrokerPlanPort loads live broker data required for trading plan builds.
type BrokerPlanPort interface {
	GetTradingSnapshot(ctx context.Context, p domain.Portfolio) (trading.BrokerSnapshot, []trading.BrokerOperation, error)
}

// CouponValueEnricher fills missing coupon amounts from an external schedule.
type CouponValueEnricher interface {
	EnrichCouponValues(bs []bonds.BondRecord, focusISINs []string)
}

// HeldUniverseAugmenter keeps existing portfolio ISINs in the market universe.
type HeldUniverseAugmenter interface {
	AugmentUniverseForISINs(universe []bonds.BondRecord, isins []string, keyRate, taxRate float64) []bonds.BondRecord
}

// PlanUseCase is the single orchestration entry for portfolio plan builds.
type PlanUseCase struct {
	repo    domain.Repository
	broker  BrokerPlanPort
	coupons CouponValueEnricher
	held    HeldUniverseAugmenter
}

func NewPlanUseCase(repo domain.Repository, broker BrokerPlanPort, coupons CouponValueEnricher, held HeldUniverseAugmenter) *PlanUseCase {
	return &PlanUseCase{repo: repo, broker: broker, coupons: coupons, held: held}
}

func (u *PlanUseCase) augmentForPositions(universe []bonds.BondRecord, positions []domain.PortfolioPosition, keyRate, taxRate float64) []bonds.BondRecord {
	if u.held == nil {
		return universe
	}
	return u.held.AugmentUniverseForISINs(universe, focusISINsFromPositions(positions), keyRate, taxRate)
}

func (u *PlanUseCase) enrichCoupons(universe []bonds.BondRecord, focusISINs []string) {
	if u.coupons == nil || len(universe) == 0 || len(focusISINs) == 0 {
		return
	}
	u.coupons.EnrichCouponValues(universe, focusISINs)
}

func focusISINsFromPositions(positions []domain.PortfolioPosition) []string {
	seen := make(map[string]struct{}, len(positions))
	out := make([]string, 0, len(positions))
	for _, pos := range positions {
		if pos.ISIN == "" {
			continue
		}
		if _, ok := seen[pos.ISIN]; ok {
			continue
		}
		seen[pos.ISIN] = struct{}{}
		out = append(out, pos.ISIN)
	}
	return out
}

func mergeFocusISINs(base []string, extra ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	add := func(isin string) {
		if isin == "" {
			return
		}
		if _, ok := seen[isin]; ok {
			return
		}
		seen[isin] = struct{}{}
		out = append(out, isin)
	}
	for _, isin := range base {
		add(isin)
	}
	for _, isin := range extra {
		add(isin)
	}
	return out
}

func (u *PlanUseCase) Build(
	ctx context.Context,
	portfolioID string,
	universe []bonds.BondRecord,
	today time.Time,
	keyRate, taxRate float64,
	assumeBestPutOutcome bool,
	durationPolicy *domain.DurationPolicy,
) (domain.PortfolioPlan, error) {
	owner, ok := auth.OwnerTelegramID(ctx)
	var p *domain.Portfolio
	var err error
	if ok {
		p, err = u.repo.GetByIDForOwner(ctx, portfolioID, owner)
	} else {
		p, err = u.repo.GetByID(ctx, portfolioID)
	}
	if err != nil || p == nil {
		return domain.PortfolioPlan{}, fmt.Errorf("%w: %s", ErrNotFound, portfolioID)
	}
	policy := durationPolicyOrDefault(*p, durationPolicy)
	if p.IsTrading() {
		if u.broker == nil {
			return domain.PortfolioPlan{}, fmt.Errorf("trading plan requires broker snapshot")
		}
		snapshot, ops, err := u.broker.GetTradingSnapshot(ctx, *p)
		if err != nil {
			return domain.PortfolioPlan{}, err
		}
		return u.buildTradingPlan(ctx, *p, snapshot, ops, universe, today, keyRate, taxRate, policy)
	}
	universe = u.augmentForPositions(universe, p.Positions, keyRate, taxRate)
	u.enrichCoupons(universe, focusISINsFromPositions(p.Positions))
	planCtx := domain.NewSimulationPlanContext(*p, assumeBestPutOutcome)
	plan := domain.BuildPlan(*p, universe, today, keyRate, taxRate, planCtx, policy)
	if _, err := u.repo.Save(ctx, *p); err != nil {
		return domain.PortfolioPlan{}, err
	}
	return plan, nil
}

func (u *PlanUseCase) BuildForTrading(
	p domain.Portfolio,
	snapshot trading.BrokerSnapshot,
	ops []trading.BrokerOperation,
	universe []bonds.BondRecord,
	today time.Time,
	keyRate, taxRate float64,
	durationPolicy domain.DurationPolicy,
) domain.PortfolioPlan {
	return u.buildTradingPlanSnapshot(p, snapshot, ops, universe, today, keyRate, taxRate, durationPolicy)
}

func (u *PlanUseCase) buildTradingPlan(
	ctx context.Context,
	p domain.Portfolio,
	snapshot trading.BrokerSnapshot,
	ops []trading.BrokerOperation,
	universe []bonds.BondRecord,
	today time.Time,
	keyRate, taxRate float64,
	durationPolicy domain.DurationPolicy,
) (domain.PortfolioPlan, error) {
	plan := u.buildTradingPlanSnapshot(p, snapshot, ops, universe, today, keyRate, taxRate, durationPolicy)
	if _, err := u.repo.Save(ctx, p); err != nil {
		return domain.PortfolioPlan{}, err
	}
	return plan, nil
}

func (u *PlanUseCase) buildTradingPlanSnapshot(
	p domain.Portfolio,
	snapshot trading.BrokerSnapshot,
	ops []trading.BrokerOperation,
	universe []bonds.BondRecord,
	today time.Time,
	keyRate, taxRate float64,
	durationPolicy domain.DurationPolicy,
) domain.PortfolioPlan {
	focus := focusISINsFromPositions(p.Positions)
	for _, h := range trading.BuildHoldings(snapshot, universe) {
		focus = mergeFocusISINs(focus, h.ISIN)
	}
	u.enrichCoupons(universe, focus)

	positions := trading.EffectiveTradingPositions(p, snapshot, universe, today)
	historical := trading.OperationsToCashflowEvents(ops, today)
	brokerCash := float64(snapshot.MoneyRub)
	historical, delta, largeNote := trading.ReconcileCashToBroker(historical, today, brokerCash)
	perfPortfolio := p
	perfPortfolio.Positions = positions
	invested := float64(trading.EpisodeCapital(perfPortfolio, snapshot, ops))
	planCtx := domain.PlanContext{
		Mode:                 domain.PlanModeTrading,
		Positions:            positions,
		HistoricalEvents:     historical,
		BrokerCashRub:        brokerCash,
		InvestedCapitalRub:   invested,
		AssumeBestPutOutcome: false,
	}
	plan := domain.BuildPlan(p, universe, today, keyRate, taxRate, planCtx, durationPolicy)
	if largeNote {
		plan.Notes = append(plan.Notes, fmt.Sprintf(
			"Сверка с брокером: расхождение %.0f ₽ (операции за lookback могут не покрывать всю историю счёта).",
			delta,
		))
	}
	return plan
}

// PlanForSlotValidation builds the same plan used for slot override validation.
func (u *PlanUseCase) PlanForSlotValidation(
	ctx context.Context,
	p domain.Portfolio,
	universe []bonds.BondRecord,
	today time.Time,
	keyRate, taxRate float64,
	durationPolicy domain.DurationPolicy,
) (domain.PortfolioPlan, error) {
	if p.IsTrading() {
		if u.broker == nil {
			return domain.PortfolioPlan{}, fmt.Errorf("trading plan requires broker snapshot")
		}
		snapshot, ops, err := u.broker.GetTradingSnapshot(ctx, p)
		if err != nil {
			return domain.PortfolioPlan{}, err
		}
		return u.buildTradingPlanSnapshot(p, snapshot, ops, universe, today, keyRate, taxRate, durationPolicy), nil
	}
	universe = u.augmentForPositions(universe, p.Positions, keyRate, taxRate)
	u.enrichCoupons(universe, focusISINsFromPositions(p.Positions))
	planCtx := domain.NewSimulationPlanContext(p, true)
	return domain.BuildPlan(p, universe, today, keyRate, taxRate, planCtx, durationPolicy), nil
}
