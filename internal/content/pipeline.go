package content

import "context"

// Parser parses raw content into a structured document.
type Parser interface {
	Parse(raw []byte, sourceType string) (*Document, error)
	SupportedTypes() []string
}

// Chunker splits documents into indexable chunks.
type Chunker interface {
	Chunk(doc *Document) ([]*Chunk, error)
}

// Indexer writes content chunks to the search index.
type Indexer interface {
	Index(ctx context.Context, chunks []*Chunk) error
	Delete(ctx context.Context, documentID string) error
}

// IngestService imports content into durable metadata and search indexes.
type IngestService interface {
	Ingest(ctx context.Context, req *IngestRequest) error
}

// IngestRequest describes a content import request.
type IngestRequest struct {
	SourceType string `json:"source_type"`
	Subject    string `json:"subject"`
	FilePath   string `json:"file_path,omitempty"`
	RawContent []byte `json:"raw_content,omitempty"`
}
