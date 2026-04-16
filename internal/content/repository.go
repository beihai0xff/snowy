package content

import (
	"context"

	"github.com/google/uuid"
)

// DocumentRepository 文档元数据持久化端口。
type DocumentRepository interface {
	Save(ctx context.Context, doc *Document) error
	GetByID(ctx context.Context, id uuid.UUID) (*Document, error)
	GetByDocID(ctx context.Context, docID string) (*Document, error)
	ListBySubject(ctx context.Context, subject string, offset, limit int) ([]*Document, int64, error)
}

// ChunkRepository 切片持久化端口。
type ChunkRepository interface {
	SaveBatch(ctx context.Context, chunks []*Chunk) error
	GetByDocumentID(ctx context.Context, documentID uuid.UUID) ([]*Chunk, error)
}
