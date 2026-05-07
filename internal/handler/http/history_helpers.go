package http

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/pkg/common"
	"github.com/beihai0xff/snowy/internal/user"
)

func recordHistory(c *gin.Context, userSvc user.Service, actionType, query string) {
	if userSvc == nil || strings.TrimSpace(query) == "" {
		return
	}

	userID := common.UserIDFromContext(c.Request.Context())
	if userID == "" {
		userID = common.DefaultUserID
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "skip history with invalid user id", "error", err)

		return
	}

	if _, err := userSvc.EnsureAnonymousUser(c.Request.Context()); err != nil && userID == common.DefaultUserID {
		slog.WarnContext(c.Request.Context(), "ensure anonymous user before history failed", "error", err)

		return
	}

	if err := userSvc.AddHistory(c.Request.Context(), &user.HistoryItem{
		UserID:     uid,
		ActionType: actionType,
		Query:      strings.TrimSpace(query),
	}); err != nil {
		slog.WarnContext(c.Request.Context(), "record history failed", "action_type", actionType, "error", err)
	}
}
