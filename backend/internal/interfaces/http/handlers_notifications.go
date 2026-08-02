package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tonatos/instrumenta/backend/internal/interfaces/auth"
)

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolio_id")
	if !h.gateTradingPortfolioID(w, r, portfolioID) {
		return
	}
	if h.deps.Portfolios != nil {
		if p, err := h.deps.Portfolios.GetPortfolio(r.Context(), portfolioID); err != nil || p == nil {
			WriteNotFound(w, "Portfolio not found")
			return
		}
	}
	unreadOnly := r.URL.Query().Get("unread_only") == "true"
	records, err := h.deps.Notifications.ListForPortfolio(r.Context(), portfolioID, unreadOnly)
	if err != nil {
		WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]NotificationResponse, 0, len(records))
	for _, record := range records {
		out = append(out, NotificationToResponse(record))
	}
	WriteJSON(w, http.StatusOK, NotificationsListResponse{Notifications: out})
}

func (h *Handler) ListOwnerNotifications(w http.ResponseWriter, r *http.Request) {
	if h.deps.Notifications == nil {
		WriteJSON(w, http.StatusOK, NotificationsListResponse{Notifications: []NotificationResponse{}})
		return
	}
	owner, ok := auth.OwnerTelegramID(r.Context())
	if !ok {
		WriteClientError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	records, err := h.deps.Notifications.ListForOwner(r.Context(), owner)
	if err != nil {
		WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]NotificationResponse, 0, len(records))
	unread := 0
	for _, record := range records {
		out = append(out, NotificationToResponse(record))
		if record.IsUnread {
			unread++
		}
	}
	WriteJSON(w, http.StatusOK, NotificationsListResponse{
		Notifications: out,
		UnreadCount:   unread,
	})
}

func (h *Handler) gateOwnedNotification(w http.ResponseWriter, r *http.Request, notificationID string) bool {
	if h.deps.Notifications == nil {
		WriteNotFound(w, "Notification not found")
		return false
	}
	record, err := h.deps.Notifications.GetByID(r.Context(), notificationID)
	if err != nil {
		WriteClientError(w, http.StatusBadRequest, err.Error())
		return false
	}
	if record == nil {
		WriteNotFound(w, "Notification not found")
		return false
	}
	if h.deps.Portfolios != nil {
		if p, err := h.deps.Portfolios.GetPortfolio(r.Context(), record.PortfolioID); err != nil || p == nil {
			WriteNotFound(w, "Notification not found")
			return false
		}
	}
	return true
}

func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	notificationID := chi.URLParam(r, "notification_id")
	if !h.gateOwnedNotification(w, r, notificationID) {
		return
	}
	record, err := h.deps.Notifications.MarkRead(r.Context(), notificationID)
	if err != nil {
		WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}
	if record == nil {
		WriteNotFound(w, "Notification not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DismissNotification(w http.ResponseWriter, r *http.Request) {
	notificationID := chi.URLParam(r, "notification_id")
	if !h.gateOwnedNotification(w, r, notificationID) {
		return
	}
	record, err := h.deps.Notifications.Dismiss(r.Context(), notificationID)
	if err != nil {
		WriteClientError(w, http.StatusBadRequest, err.Error())
		return
	}
	if record == nil {
		WriteNotFound(w, "Notification not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
