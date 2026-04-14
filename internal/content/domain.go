// Package content 定义内容入库域的领域模型与接口。
// 有界上下文：Content Ingestion — 导入、清洗、切片、标签化、建索引。
// 参考技术方案 §12。
package content

import (
	"time"

	"github.com/google/uuid"
)

// SourceType 内容来源类型。
type SourceType string

const (
	SourceTextbook SourceType = "textbook"
	SourceExamSpec SourceType = "exam_spec"
	SourceExercise SourceType = "exercise"
	SourceLecture  SourceType = "lecture"
)

// Document 原始文档元信息，参考技术方案 §12.2。
type Document struct {
	ID              uuid.UUID      `json:"id"`
	DocID           string         `json:"doc_id"`
	SourceType      SourceType     `json:"source_type"`
	Subject         string         `json:"subject"`
	Grade           string         `json:"grade"`
	Chapter         string         `json:"chapter"`
	Section         string         `json:"section"`
	KnowledgeTags   []string       `json:"knowledge_tags"`
	TopicTags       []string       `json:"topic_tags"`
	Difficulty      string         `json:"difficulty,omitempty"`
	Content         string         `json:"content"`
	Answer          string         `json:"answer,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CopyrightStatus string         `json:"copyright_status"`
	Version         int            `json:"version"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Chunk 文档切片。
type Chunk struct {
	ID         uuid.UUID `json:"id"`
	DocumentID uuid.UUID `json:"document_id"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	Embedding  []float64 `json:"-"`
	Tags       []string  `json:"tags,omitempty"`
	ChunkType  string    `json:"chunk_type"` // paragraph / section / question / concept / process
	CreatedAt  time.Time `json:"created_at"`
}
