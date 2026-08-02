import type { Notification, NotificationKind } from "@/api/types";
import { sectorLabel } from "@/features/bonds/sectorLabels";
import { isMarketSignalKind } from "@/features/portfolio/marketSignals";
import { NOTIFICATION_KIND_LABELS } from "@/features/portfolio/labels";
import { formatPct } from "@/lib/utils";

const SECTOR_KEY_RE =
  /\b(unknown|financial|real_estate|utilities|it|telecom|consumer|materials|industrials|energy|government|health_care|other|corp)\b/gi;

export function isImportantNotification(notification: Notification): boolean {
  return !isMarketSignalKind(notification.kind);
}

export function notificationKindLabel(notification: Notification): string {
  return NOTIFICATION_KIND_LABELS[notification.kind] ?? notification.kind;
}

export function humanizeSectorText(text: string): string {
  return text.replace(SECTOR_KEY_RE, (raw) => sectorLabel(raw));
}

export function notificationTitle(notification: Notification): string {
  const payload = notification.payload ?? {};
  const name = typeof payload.name === "string" ? payload.name : "Уведомление";
  if (notification.kind === "sector_concentration") {
    return humanizeSectorText(name);
  }
  return name;
}

export function notificationBody(notification: Notification): string {
  const reason = notification.payload?.reason;
  if (typeof reason !== "string" || reason.length === 0) return "";
  if (notification.kind === "sector_concentration") {
    return humanizeSectorText(reason);
  }
  return reason;
}

export function notificationSourceLabel(notification: Notification): string {
  const portfolio = notification.portfolio_name?.trim() || "Портфель";
  if (isMarketSignalKind(notification.kind)) {
    return `Радар · ${portfolio}`;
  }
  return portfolio;
}

export function notificationBorderClass(notification: Notification): string {
  if (!notification.is_unread) {
    return "border-border/60";
  }
  if (notification.urgency === "critical" || notification.kind === "risk_escalation") {
    return "border-red-400/50";
  }
  if (notification.urgency === "soon" || notification.kind === "put_offer_action") {
    return "border-amber-400/40";
  }
  if (isMarketSignalKind(notification.kind)) {
    return "border-sky-400/40";
  }
  return "border-border/60";
}

export function notificationBackgroundClass(notification: Notification): string {
  if (!notification.is_unread) {
    return "bg-card/50";
  }
  if (notification.urgency === "critical" || notification.kind === "risk_escalation") {
    return "bg-red-500/5";
  }
  if (notification.urgency === "soon" || notification.kind === "put_offer_action") {
    return "bg-amber-500/10";
  }
  if (isMarketSignalKind(notification.kind)) {
    return "bg-sky-500/5";
  }
  return "bg-card/50";
}

export function notificationKindTone(kind: NotificationKind | string): string {
  switch (kind) {
    case "risk_escalation":
      return "text-red-500";
    case "put_offer_action":
      return "text-amber-500";
    case "put_offer_watch":
      return "text-amber-500/80";
    case "sector_concentration":
      return "text-violet-500";
    case "spread_anomaly":
    case "spread_widening":
      return "text-sky-500";
    case "sector_stress":
      return "text-orange-500";
    case "turbo_entry":
      return "text-emerald-500";
    default:
      return "text-muted-foreground";
  }
}

export function payloadPct(payload: Record<string, unknown>, key: string): string | null {
  const v = payload[key];
  if (typeof v !== "number" || !Number.isFinite(v)) return null;
  return formatPct(v);
}
