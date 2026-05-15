package user

import (
	"time"

	"github.com/google/uuid"
)

const (
	ReactionLike    = "like"
	ReactionDislike = "dislike"

	ReactionVisibilityPublic  = "public"
	ReactionVisibilityPrivate = "private"
)

// Reaction stores one user's quality feedback for an answer/evidence/model package/render artifact.
type Reaction struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	ReactionType string    `json:"reaction_type"`
	Visibility   string    `json:"visibility"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ReactionUserSummary is a privacy-aware user summary exposed for public reactions.
type ReactionUserSummary struct {
	ID       uuid.UUID `json:"id"`
	Nickname string    `json:"nickname"`
	Role     Role      `json:"role"`
}

// ReactionSummary aggregates community feedback for a target.
type ReactionSummary struct {
	TargetType      string                `json:"target_type"`
	TargetID        string                `json:"target_id"`
	LikeCount       int64                 `json:"like_count"`
	DislikeCount    int64                 `json:"dislike_count"`
	CurrentReaction string                `json:"current_reaction,omitempty"`
	LikeUsers       []ReactionUserSummary `json:"like_users,omitempty"`
	DislikeUsers    []ReactionUserSummary `json:"dislike_users,omitempty"`
}
