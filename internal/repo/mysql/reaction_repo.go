package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	searchdomain "github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/beihai0xff/snowy/internal/user"
)

type reactionRepo struct{ db *gorm.DB }

func NewReactionRepository(db *gorm.DB) *reactionRepo {
	return &reactionRepo{db: db}
}

func (r *reactionRepo) Upsert(ctx context.Context, reaction *user.Reaction) error {
	if reaction == nil {
		return errors.New("reaction is nil")
	}

	row := newReactionRow(reaction)
	err := dbFromContext(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "target_type"}, {Name: "target_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"reaction_type": row.ReactionType,
			"visibility":    row.Visibility,
			"updated_at":    row.UpdatedAt,
		}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("upsert reaction: %w", err)
	}

	return nil
}

func (r *reactionRepo) Delete(ctx context.Context, userID uuid.UUID, targetType string, targetID string) error {
	result := dbFromContext(ctx, r.db).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&reactionRow{})
	if result.Error != nil {
		return fmt.Errorf("delete reaction: %w", result.Error)
	}

	return nil
}

func (r *reactionRepo) ListByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*user.Reaction, int64, error) {
	return listByUserRows[reactionRow](ctx, r.db, &reactionRow{}, userID, offset, limit,
		"updated_at DESC",
		"reactions", "reactions",
		func(row *reactionRow) (*user.Reaction, error) { return row.toDomain(), nil },
	)
}

func (r *reactionRepo) TargetFeedback(ctx context.Context, targetType string, targetID string) (searchdomain.FeedbackSummary, error) {
	var summary searchdomain.FeedbackSummary
	gdb := dbFromContext(ctx, r.db)

	if err := gdb.Model(&reactionRow{}).
		Where("target_type = ? AND target_id = ? AND reaction_type = ?", targetType, targetID, user.ReactionLike).
		Count(&summary.LikeCount).Error; err != nil {
		return summary, fmt.Errorf("count feedback likes: %w", err)
	}

	if err := gdb.Model(&reactionRow{}).
		Where("target_type = ? AND target_id = ? AND reaction_type = ?", targetType, targetID, user.ReactionDislike).
		Count(&summary.DislikeCount).Error; err != nil {
		return summary, fmt.Errorf("count feedback dislikes: %w", err)
	}

	return summary, nil
}

func (r *reactionRepo) Summary(
	ctx context.Context,
	userID uuid.UUID,
	targetType string,
	targetID string,
	includeUsers bool,
) (*user.ReactionSummary, error) {
	gdb := dbFromContext(ctx, r.db)
	summary := &user.ReactionSummary{TargetType: targetType, TargetID: targetID}

	if err := gdb.Model(&reactionRow{}).
		Where("target_type = ? AND target_id = ? AND reaction_type = ?", targetType, targetID, user.ReactionLike).
		Count(&summary.LikeCount).Error; err != nil {
		return nil, fmt.Errorf("count likes: %w", err)
	}

	if err := gdb.Model(&reactionRow{}).
		Where("target_type = ? AND target_id = ? AND reaction_type = ?", targetType, targetID, user.ReactionDislike).
		Count(&summary.DislikeCount).Error; err != nil {
		return nil, fmt.Errorf("count dislikes: %w", err)
	}

	if userID != uuid.Nil {
		var current reactionRow
		err := gdb.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).Take(&current).Error
		if err == nil {
			summary.CurrentReaction = current.ReactionType
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("get current reaction: %w", err)
		}
	}

	if includeUsers {
		likes, err := r.reactionUsers(ctx, targetType, targetID, user.ReactionLike)
		if err != nil {
			return nil, err
		}
		dislikes, err := r.reactionUsers(ctx, targetType, targetID, user.ReactionDislike)
		if err != nil {
			return nil, err
		}
		summary.LikeUsers = likes
		summary.DislikeUsers = dislikes
	}

	return summary, nil
}

func (r *reactionRepo) reactionUsers(
	ctx context.Context,
	targetType string,
	targetID string,
	reactionType string,
) ([]user.ReactionUserSummary, error) {
	type row struct {
		ID       uuid.UUID `gorm:"column:id"`
		Nickname string    `gorm:"column:nickname"`
		Role     user.Role `gorm:"column:role"`
	}

	rows := []row{}
	err := dbFromContext(ctx, r.db).Table("reactions").
		Select("users.id, users.nickname, users.role").
		Joins("JOIN users ON users.id = reactions.user_id").
		Where("reactions.target_type = ? AND reactions.target_id = ? AND reactions.reaction_type = ? AND reactions.visibility = ?", targetType, targetID, reactionType, user.ReactionVisibilityPublic).
		Order("reactions.updated_at DESC").
		Limit(20).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list reaction users: %w", err)
	}

	out := make([]user.ReactionUserSummary, 0, len(rows))
	for _, item := range rows {
		out = append(out, user.ReactionUserSummary{ID: item.ID, Nickname: item.Nickname, Role: item.Role})
	}

	return out, nil
}
