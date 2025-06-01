package pgvector

type PGVectorOption func(s *PGVectorStorage)

func WithPartitionsEnabled(enabled bool) PGVectorOption {
	return func(s *PGVectorStorage) {
		s.partitionsEnabled = enabled
	}
}

func WithEmbeddingVectorDimensions(dimensions uint32) PGVectorOption {
	return func(s *PGVectorStorage) {
		s.embeddingVectorDimensions = dimensions
	}
}

func WithDatabaseName(databaseName string) PGVectorOption {
	return func(s *PGVectorStorage) {
		s.databaseName = databaseName
	}
}

func WithDatabaseSchema(databaseSchema string) PGVectorOption {
	return func(s *PGVectorStorage) {
		s.databaseSchema = databaseSchema
	}
}

func WithDatabasePrefix(databasePrefix string) PGVectorOption {
	return func(s *PGVectorStorage) {
		s.databasePrefix = databasePrefix
	}
}
