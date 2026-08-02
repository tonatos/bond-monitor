import {
  forwardRef,
  useState,
  type ComponentPropsWithoutRef,
  type ReactNode,
} from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  Activity,
  AlertTriangle,
  Bell,
  CalendarClock,
  Eye,
  PieChart,
  TrendingDown,
  Zap,
  type LucideIcon,
} from "lucide-react";
import { api } from "@/api/client";
import type { Notification, NotificationKind } from "@/api/types";
import {
  isImportantNotification,
  notificationBody,
  notificationKindLabel,
  notificationKindTone,
  notificationSourceLabel,
  notificationTitle,
  payloadPct,
} from "@/features/notifications/notificationDisplay";
import {
  notificationsInboxQueryKey,
  useNotificationsInbox,
} from "@/features/portfolio/marketSignals";
import { Button } from "@/components/ui/button";
import { PopoverContent, PopoverRoot, PopoverTrigger } from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { cn, formatDateTime } from "@/lib/utils";

const KIND_ICONS: Record<NotificationKind, LucideIcon> = {
  put_offer_action: CalendarClock,
  put_offer_watch: Eye,
  risk_escalation: AlertTriangle,
  sector_concentration: PieChart,
  spread_anomaly: Activity,
  spread_widening: TrendingDown,
  sector_stress: TrendingDown,
  turbo_entry: Zap,
};

function InboxList({
  notifications,
  onOpenNotification,
  onMarkRead,
  markReadPending,
}: {
  notifications: Notification[];
  onOpenNotification: (notification: Notification) => void;
  onMarkRead: (id: string) => void;
  markReadPending: boolean;
}) {
  if (notifications.length === 0) {
    return (
      <p className="py-8 text-center text-sm text-muted-foreground">
        Уведомлений пока нет
      </p>
    );
  }

  return (
    <div data-testid="notifications-inbox-list">
      {notifications.map((notification, index) => {
        const payload = notification.payload ?? {};
        const bond7d = payloadPct(payload, "bond_change_7d_pct");
        const sector7d = payloadPct(payload, "sector_change_7d_pct");
        const important = isImportantNotification(notification);
        const body = notificationBody(notification);
        const canOpen = Boolean(notification.portfolio_id);
        const Icon = KIND_ICONS[notification.kind] ?? Bell;

        return (
          <div key={notification.id}>
            {index > 0 ? <Separator /> : null}
            <div
              data-testid={`inbox-notification-${notification.id}`}
              className={cn(
                "flex min-w-0 gap-3 py-3",
                canOpen && "cursor-pointer hover:bg-accent/40",
                !notification.is_unread && "opacity-70",
              )}
              onClick={() => onOpenNotification(notification)}
              onKeyDown={(e) => {
                if (!canOpen) return;
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onOpenNotification(notification);
                }
              }}
              role={canOpen ? "link" : undefined}
              tabIndex={canOpen ? 0 : undefined}
            >
              <span
                className={cn(
                  "mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted/60",
                  notificationKindTone(notification.kind),
                )}
                aria-hidden
              >
                <Icon className="h-4 w-4" />
              </span>
              <div className="min-w-0 flex-1 space-y-1.5">
                <div className="flex min-w-0 items-start justify-between gap-2">
                  <p
                    className={cn(
                      "min-w-0 break-words text-sm",
                      notification.is_unread ? "font-semibold" : "font-medium",
                    )}
                  >
                    {notificationTitle(notification)}
                  </p>
                  <p className="shrink-0 text-[11px] leading-5 text-muted-foreground">
                    {formatDateTime(notification.created_at)}
                  </p>
                </div>
                <p className="truncate text-xs text-muted-foreground">
                  {notificationSourceLabel(notification)}
                  <span className="mx-1 text-border">·</span>
                  {notificationKindLabel(notification)}
                </p>
                {body ? (
                  <p className="break-words text-sm text-muted-foreground">{body}</p>
                ) : null}
                {(bond7d || sector7d) && (
                  <p className="break-words text-xs text-muted-foreground">
                    {bond7d && <span>Бумага 7д: {bond7d}</span>}
                    {bond7d && sector7d && <span> · </span>}
                    {sector7d && <span>Сектор 7д: {sector7d}</span>}
                  </p>
                )}
                {important && notification.is_unread ? (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-1 min-h-10"
                    disabled={markReadPending}
                    onClick={(e) => {
                      e.stopPropagation();
                      onMarkRead(notification.id);
                    }}
                  >
                    Прочитано
                  </Button>
                ) : null}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

const BellTriggerButton = forwardRef<
  HTMLButtonElement,
  ComponentPropsWithoutRef<typeof Button> & { unreadCount: number }
>(function BellTriggerButton({ unreadCount, className, ...props }, ref) {
  return (
    <Button
      ref={ref}
      type="button"
      variant="outline"
      size="icon"
      className={cn("relative min-h-10 min-w-10", className)}
      aria-label="Уведомления"
      data-testid="notifications-bell"
      {...props}
    >
      <Bell className="h-4 w-4" />
      {unreadCount > 0 ? (
        <span
          className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-medium text-primary-foreground"
          data-testid="notifications-bell-badge"
        >
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      ) : null}
    </Button>
  );
});

function InboxBody({
  isLoading,
  list,
  className,
}: {
  isLoading: boolean;
  list: ReactNode;
  className?: string;
}) {
  return (
    <ScrollArea className={cn("w-full", className)} data-testid="notifications-inbox-scroll">
      <div className="box-border w-full min-w-0 px-4 py-1">
        {isLoading ? (
          <p className="py-8 text-center text-sm text-muted-foreground">Загрузка…</p>
        ) : (
          list
        )}
      </div>
    </ScrollArea>
  );
}

export function NotificationBell({ mode }: { mode: "popover" | "sheet" }) {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data, isLoading } = useNotificationsInbox();
  const notifications = data?.notifications ?? [];
  const unreadCount =
    data?.unread_count ?? notifications.filter((n) => n.is_unread).length;

  const markRead = useMutation({
    mutationFn: (notificationId: string) => api.markNotificationRead(notificationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notificationsInboxQueryKey });
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  const openNotification = (notification: Notification) => {
    if (notification.is_unread && !isImportantNotification(notification)) {
      markRead.mutate(notification.id);
    }
    if (notification.portfolio_id) {
      setOpen(false);
      navigate(`/portfolio/${encodeURIComponent(notification.portfolio_id)}`);
    }
  };

  const list = (
    <InboxList
      notifications={notifications}
      onOpenNotification={openNotification}
      onMarkRead={(id) => markRead.mutate(id)}
      markReadPending={markRead.isPending}
    />
  );

  if (mode === "sheet") {
    return (
      <>
        <BellTriggerButton
          unreadCount={unreadCount}
          aria-expanded={open}
          onClick={() => setOpen(true)}
        />
        <Sheet open={open} onOpenChange={setOpen}>
          <SheetContent
            side="right"
            className="flex w-full max-w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-md"
            data-testid="notifications-inbox-sheet"
          >
            <SheetHeader className="shrink-0 border-b border-border px-4 py-3 pr-12 text-left">
              <SheetTitle className="text-base">Уведомления</SheetTitle>
            </SheetHeader>
            <InboxBody isLoading={isLoading} list={list} className="min-h-0 flex-1" />
          </SheetContent>
        </Sheet>
      </>
    );
  }

  return (
    <PopoverRoot open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <BellTriggerButton unreadCount={unreadCount} />
      </PopoverTrigger>
      <PopoverContent
        side="top"
        align="start"
        sideOffset={12}
        collisionPadding={0}
        avoidCollisions={false}
        className="z-[60] w-[min(24rem,calc(100vw-2rem))] overflow-hidden p-0"
        data-testid="notifications-inbox-popover"
      >
        <div className="border-b border-border px-4 py-3">
          <p className="text-sm font-semibold">Уведомления</p>
        </div>
        <InboxBody isLoading={isLoading} list={list} className="h-[min(28rem,70vh)]" />
      </PopoverContent>
    </PopoverRoot>
  );
}
