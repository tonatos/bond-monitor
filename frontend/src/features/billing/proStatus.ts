import type { BillingStatusResponse } from "@/api/types";
import { formatDate } from "@/lib/utils";

export function hasProAccess(status: BillingStatusResponse | undefined | null): boolean {
  return Boolean(status?.has_active_access || status?.complimentary);
}

export function periodLabel(period: string | undefined | null): string {
  if (period === "year") return "Год";
  if (period === "month") return "Месяц";
  return period ?? "";
}

/** Compact chrome label: «Pro», «Pro · до 1 янв». */
export function proBadgeText(opts: {
  endAt?: string | null;
  complimentary?: boolean;
  compact?: boolean;
}): string {
  if (opts.compact) return "Pro";
  if (opts.endAt) return `Pro · до ${formatDate(opts.endAt)}`;
  if (opts.complimentary) return "Pro";
  return "Pro";
}

/** Accessible full phrase for aria-label / title. */
export function proBadgeAriaLabel(opts: {
  endAt?: string | null;
  complimentary?: boolean;
}): string {
  if (opts.endAt) return `Pro до ${formatDate(opts.endAt)}`;
  if (opts.complimentary) return "Pro, complimentary-доступ";
  return "Pro";
}

/** Hero line on PlanPage for active access. */
export function proActiveHeadline(opts: {
  endAt?: string | null;
  complimentary?: boolean;
}): string {
  if (opts.complimentary && !opts.endAt) return "Complimentary‑доступ";
  if (opts.endAt) return `Активна до ${formatDate(opts.endAt)}`;
  return "Активна";
}

export function renewalHint(opts: {
  recurringEnabled: boolean;
  cancelAtPeriodEnd: boolean;
}): string | null {
  if (opts.recurringEnabled && opts.cancelAtPeriodEnd) {
    return "Автопродление отключено";
  }
  if (!opts.recurringEnabled) {
    return "Без автопродления — оплата периода вручную";
  }
  return null;
}
