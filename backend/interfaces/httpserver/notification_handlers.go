package httpserver

import (
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/usecase/inbox"
)

type notificationResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Message   string            `json:"message"`
	Params    map[string]string `json:"params,omitempty"`
	Link      string            `json:"link,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	Read      bool              `json:"read"`
}

type notificationListResponse struct {
	Items  []notificationResponse `json:"items"`
	Unread int                    `json:"unread"`
}

type unreadCountResponse struct {
	Unread int `json:"unread"`
}

func handleListNotifications(uc *inbox.List) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		out, err := uc.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		items := make([]notificationResponse, 0, len(out.Items))
		for _, n := range out.Items {
			items = append(items, notificationResponse{
				ID: n.ID.Hex(), Type: n.Type, Message: n.Message, Params: n.Params, Link: n.Link,
				CreatedAt: n.CreatedAt, Read: n.ReadAt != nil,
			})
		}
		writeJSON(w, http.StatusOK, notificationListResponse{Items: items, Unread: out.Unread})
	}
}

func handleUnreadNotifications(uc *inbox.UnreadCount) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		n, err := uc.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, unreadCountResponse{Unread: n})
	}
}

func handleMarkNotificationRead(uc *inbox.MarkRead) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		if err := uc.Execute(r.Context(), claims.UserID, id); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleMarkAllNotificationsRead(uc *inbox.MarkAllRead) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		if err := uc.Execute(r.Context(), claims.UserID); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
