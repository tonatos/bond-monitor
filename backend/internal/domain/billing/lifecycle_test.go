package billing_test

import (
	"testing"
	"time"

	"github.com/tonatos/instrumenta/backend/internal/domain/billing"
)

func TestApplySuccessfulPayment_RecurringSavesMethod(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	plan := billing.PlanVersion{
		ID:            "pro_month_v1",
		Period:        billing.PeriodMonth,
		AmountKopecks: 79500,
		Features:      billing.PaidFeaturesV1(),
	}
	sub := billing.ApplySuccessfulPayment(nil, plan, 10, "checkout", now, "pm_1", true)
	if sub.PaymentMethodID != "pm_1" {
		t.Fatalf("payment method: %q", sub.PaymentMethodID)
	}
	if sub.CancelAtPeriodEnd {
		t.Fatal("recurring checkout must keep auto-renew")
	}
	if !billing.ShouldAttemptRenew(sub, now.Add(31*24*time.Hour)) {
		t.Fatal("expected renew attempt after period end")
	}
}

func TestApplySuccessfulPayment_OneTimeNoAutoRenew(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	plan := billing.PlanVersion{
		ID:            "pro_month_v1",
		Period:        billing.PeriodMonth,
		AmountKopecks: 79500,
		Features:      billing.PaidFeaturesV1(),
	}
	// Even if provider returns a method id, one-time mode must not store it.
	sub := billing.ApplySuccessfulPayment(nil, plan, 10, "checkout", now, "pm_ignored", false)
	if sub.PaymentMethodID != "" {
		t.Fatalf("one-time must clear payment method, got %q", sub.PaymentMethodID)
	}
	if !sub.CancelAtPeriodEnd {
		t.Fatal("one-time period must end without auto-renew")
	}
	if billing.ShouldAttemptRenew(sub, now.Add(31*24*time.Hour)) {
		t.Fatal("one-time must not renew")
	}
	if !billing.ShouldExpireCanceled(sub, now.Add(31*24*time.Hour)) {
		t.Fatal("one-time must expire after period end")
	}
}
