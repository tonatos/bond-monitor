import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Bell } from "lucide-react";
import { api } from "@/api/client";
import {
  notificationBackgroundClass,
  notificationBody,
  notificationBorderClass,
  notificationKindLabel,
  notificationTitle,
} from "@/features/notifications/notificationDisplay";
import {
  notificationsInboxQueryKey,
  portfolioNotificationsQueryKey,
  usePortfolioNotifications,
} from "@/features/portfolio/marketSignals";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn, formatDateTime } from "@/lib/utils";

interface NotificationsPanelProps {
  portfolioId: string;
}

export function NotificationsPanel({ portfolioId }: NotificationsPanelProps) {
  const queryClient = useQueryClient();
  const { notifications, unreadNotificationsCount, isLoading } =
    usePortfolioNotifications(portfolioId);

  const markRead = useMutation({
    mutationFn: (notificationId: string) => api.markNotificationRead(notificationId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: portfolioNotificationsQueryKey(portfolioId),
      });
      void queryClient.invalidateQueries({ queryKey: notificationsInboxQueryKey });
    },
  });

  if (isLoading) {
    return null;
  }

  if (notifications.length === 0) {
    return null;
  }

  return (
    <div
      className="space-y-4 rounded-xl border border-amber-400/40 bg-amber-500/5 p-4"
      data-testid="notifications-panel"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="flex items-center gap-2 text-sm font-semibold text-amber-800 dark:text-amber-300">
          <Bell className="h-4 w-4" />
          Уведомления
          {unreadNotificationsCount > 0 && (
            <Badge
              className="bg-amber-500/20 text-amber-900 dark:text-amber-200"
              data-testid="notifications-unread-badge"
            >
              {unreadNotificationsCount}
            </Badge>
          )}
        </p>
      </div>

      <div className="space-y-2">
        {notifications.map((notification) => {
          const kindLabel = notificationKindLabel(notification);

          return (
            <div
              key={notification.id}
              data-testid={`notification-${notification.id}`}
              className={cn(
                "space-y-2 rounded-lg border p-3",
                notificationBorderClass(notification),
                notificationBackgroundClass(notification),
              )}
            >
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div className="min-w-0 space-y-0.5">
                  <p className="text-sm font-medium">{notificationTitle(notification)}</p>
                  <p
                    className="text-xs text-muted-foreground"
                    data-testid={`notification-${notification.id}-created-at`}
                  >
                    {formatDateTime(notification.created_at)}
                  </p>
                </div>
                <Badge variant="outline" className="shrink-0 text-xs">
                  {kindLabel}
                </Badge>
              </div>
              {notificationBody(notification) && (
                <p className="text-sm text-muted-foreground">{notificationBody(notification)}</p>
              )}
              {notification.is_unread && (
                <Button
                  variant="outline"
                  size="sm"
                  className="min-h-10"
                  onClick={() => markRead.mutate(notification.id)}
                  disabled={markRead.isPending}
                >
                  Прочитано
                </Button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
