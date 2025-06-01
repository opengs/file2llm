package testlib

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/opengs/file2llm/embedder/testlib"
	"github.com/opengs/file2llm/storage"
)

func RandString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func RandSchemaName(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func TestStorage(t *testing.T, s storage.Storage, dimensions int) {
	t.Run("CreateDeleteSource", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))

		source, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Error(err.Error())
			return
		}
		if source.UUID != sourceUUID {
			t.Fail()
			return
		}

		err = s.DeleteSource(t.Context(), source.UUID)
		if err != nil {
			t.Error(err.Error())
			return
		}
	})

	t.Run("CreateFileWithoutSource", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		filePath := "/some/path/file.txt"
		eTag := RandString(16)
		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "model-v1"}

		// Try to create file without creating source first
		_, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
		if err == nil {
			t.Errorf("expected error when creating file without source, got nil")
		}
	})

	t.Run("GetOrCreateSourceIdempotent", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))

		source1, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}
		source2, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		if source1.UUID != source2.UUID {
			t.Errorf("expected same source UUID, got %s and %s", source1.UUID, source2.UUID)
		}
	})

	t.Run("CreateGetFile", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		filePath := "/test/file1.txt"
		eTag := RandString(16)
		procVer := storage.ProcessorVersion{Major: 1, Minor: 2, Patch: 3, EmbeddingsModel: "embedding-model"}

		file1, created, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
		if err != nil {
			t.Fatal(err)
		}
		if !created {
			t.Error("expected file to be created, but created flag was false")
		}
		if file1.Path != filePath || file1.ETag != eTag {
			t.Errorf("file mismatch: expected path %s and etag %s but got %+v", filePath, eTag, file1)
		}

		// Try to get the same file again; should not be created
		file2, created, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
		if err != nil {
			t.Fatal(err)
		}
		if created {
			t.Error("expected file to be found, not created")
		}
		if file2.UUID != file1.UUID {
			t.Errorf("expected same file UUID %s but got %s", file1.UUID, file2.UUID)
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		filePath := "/file/delete.txt"
		eTag := RandString(16)
		procVer := storage.ProcessorVersion{Major: 0, Minor: 0, Patch: 1, EmbeddingsModel: "v1"}

		file, created, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
		if err != nil {
			t.Fatal(err)
		}
		if !created {
			t.Fatal("expected file to be created")
		}

		// Delete the file
		err = s.DeleteFile(t.Context(), sourceUUID, file.UUID)
		if err != nil {
			t.Fatal(err)
		}

		// Deleting again should error or at least return file not found error
		err = s.DeleteFile(t.Context(), sourceUUID, file.UUID)
		if err == nil {
			t.Error("expected error when deleting non-existing file, got nil")
		}
	})

	t.Run("FinishFileProcessing", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		filePath := "/file/finish.txt"
		eTag := RandString(16)
		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "v1"}

		file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
		if err != nil {
			t.Fatal(err)
		}

		partsErrors := []string{
			"missing field",
			"invalid syntax",
		}

		err = s.FinishFileProcessing(t.Context(), sourceUUID, file.UUID, true, "", partsErrors)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("PutEmbeddingAndSearch", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		filePath := "/file/embed.txt"
		eTag := RandString(16)
		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "v1"}

		file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
		if err != nil {
			t.Fatal(err)
		}

		chunk := "chunk-001"
		vector := testlib.RandNormalizedEmbedding(dimensions)

		err = s.PutEmbedding(t.Context(), sourceUUID, file.UUID, chunk, vector)
		if err != nil {
			t.Fatal(err)
		}

		// Search for similar embeddings in this source
		results, err := s.SearchSimilarEmbedddings(t.Context(), vector, []storage.SourceUUID{sourceUUID}, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) == 0 {
			t.Error("expected at least one embedding result, got none")
		}

		// Search across all sources (empty slice)
		resultsAll, err := s.SearchSimilarEmbedddings(t.Context(), vector, []storage.SourceUUID{}, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(resultsAll) == 0 {
			t.Error("expected at least one embedding result searching all sources, got none")
		}
	})

	t.Run("MultipleEmbeddingsPerFileAndSearch", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "multi-embed-model"}

		type vecEntry struct {
			vector []float32
			fileID storage.FileUUID
			chunk  string
		}

		var entries []vecEntry

		for i := 0; i < 50; i++ {
			filePath := "/file/multi/" + RandString(8)
			eTag := RandString(16)

			file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
			if err != nil {
				t.Fatalf("failed to create file %d: %v", i, err)
			}

			// Add multiple (e.g., 20) embeddings per file
			for j := 0; j < 20; j++ {
				vector := testlib.RandNormalizedEmbedding(dimensions)
				chunkID := "chunk-" + RandString(6)

				err := s.PutEmbedding(t.Context(), sourceUUID, file.UUID, chunkID, vector)
				if err != nil {
					t.Fatalf("failed to put embedding %d for file %d: %v", j, i, err)
				}

				entries = append(entries, vecEntry{
					vector: vector,
					fileID: file.UUID,
					chunk:  chunkID,
				})
			}
		}

		// Pick a random embedding to search for
		selected := entries[rand.Intn(len(entries))]

		results, err := s.SearchSimilarEmbedddings(t.Context(), selected.vector, []storage.SourceUUID{sourceUUID}, 10)
		if err != nil {
			t.Fatal(err)
		}

		if len(results) == 0 {
			t.Fatal("expected some results, got none")
		}

		found := false
		for _, res := range results {
			if res.File.UUID == selected.fileID && res.Chunk == selected.chunk {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected selected embedding to be among top 10 similar results, but it was not found")
		}
	})

	t.Run("MultiSourceEmbeddingSearch", func(t *testing.T) {
		numSources := 3
		filesPerSource := 20
		embeddingsPerFile := 3

		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "multi-source-model"}

		type vecEntry struct {
			vector []float32
			fileID storage.FileUUID
			chunk  string
			source storage.SourceUUID
		}

		var entries []vecEntry

		// Create multiple sources and populate them
		for sIdx := 0; sIdx < numSources; sIdx++ {
			sourceUUID := storage.SourceUUID(RandString(32))

			_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
			if err != nil {
				t.Fatalf("failed to create source %d: %v", sIdx, err)
			}

			for fIdx := 0; fIdx < filesPerSource; fIdx++ {
				filePath := "/source/" + RandString(6)
				eTag := RandString(16)

				file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
				if err != nil {
					t.Fatalf("failed to create file %d in source %d: %v", fIdx, sIdx, err)
				}

				for eIdx := 0; eIdx < embeddingsPerFile; eIdx++ {
					vector := testlib.RandNormalizedEmbedding(dimensions)
					chunk := "chunk-" + RandString(5)

					err := s.PutEmbedding(t.Context(), sourceUUID, file.UUID, chunk, vector)
					if err != nil {
						t.Fatalf("failed to put embedding %d in file %d in source %d: %v", eIdx, fIdx, sIdx, err)
					}

					entries = append(entries, vecEntry{
						vector: vector,
						fileID: file.UUID,
						chunk:  chunk,
						source: sourceUUID,
					})
				}
			}
		}

		// Pick a random embedding to use for search
		selected := entries[rand.Intn(len(entries))]

		// Search in the specific source only
		resultsInSource, err := s.SearchSimilarEmbedddings(t.Context(), selected.vector, []storage.SourceUUID{selected.source}, 10)
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, res := range resultsInSource {
			if res.File.UUID == selected.fileID && res.Chunk == selected.chunk {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected embedding to be found in specific source search, but it was not")
		}

		// Search across all sources
		resultsAll, err := s.SearchSimilarEmbedddings(t.Context(), selected.vector, []storage.SourceUUID{}, 10)
		if err != nil {
			t.Fatal(err)
		}

		foundAll := false
		for _, res := range resultsAll {
			if res.File.UUID == selected.fileID && res.Chunk == selected.chunk {
				foundAll = true
				break
			}
		}
		if !foundAll {
			t.Error("expected embedding to be found in global search, but it was not")
		}
	})

	t.Run("MultiSourceSearchIncludesAndExcludesCorrectly", func(t *testing.T) {
		numSources := 3
		filesPerSource := 20
		embeddingsPerFile := 3

		procVer := storage.ProcessorVersion{
			Major:           1,
			Minor:           0,
			Patch:           0,
			EmbeddingsModel: "multi-source-model",
		}

		type vecEntry struct {
			vector []float32
			fileID storage.FileUUID
			chunk  string
			source storage.SourceUUID
		}

		var entries []vecEntry
		var sourceUUIDs []storage.SourceUUID

		for sIdx := 0; sIdx < numSources; sIdx++ {
			sourceUUID := storage.SourceUUID(RandString(32))
			sourceUUIDs = append(sourceUUIDs, sourceUUID)

			_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
			if err != nil {
				t.Fatalf("failed to create source %d: %v", sIdx, err)
			}

			for fIdx := 0; fIdx < filesPerSource; fIdx++ {
				filePath := "/source/" + RandString(6)
				eTag := RandString(16)

				file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, filePath, eTag, procVer)
				if err != nil {
					t.Fatalf("failed to create file %d in source %d: %v", fIdx, sIdx, err)
				}

				for eIdx := 0; eIdx < embeddingsPerFile; eIdx++ {
					vector := testlib.RandNormalizedEmbedding(dimensions)
					chunk := "chunk-" + RandString(5)

					err := s.PutEmbedding(t.Context(), sourceUUID, file.UUID, chunk, vector)
					if err != nil {
						t.Fatalf("failed to put embedding: %v", err)
					}

					entries = append(entries, vecEntry{
						vector: vector,
						fileID: file.UUID,
						chunk:  chunk,
						source: sourceUUID,
					})
				}
			}
		}

		selected := entries[rand.Intn(len(entries))]

		// ✅ 1. Search in the correct source
		resultsCorrect, err := s.SearchSimilarEmbedddings(t.Context(), selected.vector, []storage.SourceUUID{selected.source}, 10)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, res := range resultsCorrect {
			if res.File.UUID == selected.fileID && res.Chunk == selected.chunk {
				found = true
				break
			}
		}
		if !found {
			t.Error("embedding not found in correct source search")
		}

		// ✅ 2. Search in all sources
		resultsAll, err := s.SearchSimilarEmbedddings(t.Context(), selected.vector, nil, 10)
		if err != nil {
			t.Fatal(err)
		}
		foundAll := false
		for _, res := range resultsAll {
			if res.File.UUID == selected.fileID && res.Chunk == selected.chunk {
				foundAll = true
				break
			}
		}
		if !foundAll {
			t.Error("embedding not found in global search")
		}

		// ❌ 3. Search in wrong source (pick any that is not selected.source)
		var wrongSource storage.SourceUUID
		for _, sid := range sourceUUIDs {
			if sid != selected.source {
				wrongSource = sid
				break
			}
		}

		resultsWrong, err := s.SearchSimilarEmbedddings(t.Context(), selected.vector, []storage.SourceUUID{wrongSource}, 10)
		if err != nil {
			t.Fatal(err)
		}

		for _, res := range resultsWrong {
			if res.File.Source.UUID == selected.source {
				t.Error("embedding should not be found in wrong source search")
				break
			}
		}
	})

	t.Run("DeleteSourceCleansAll", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "delete-cleanup-model"}
		file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, "/file/to/delete.txt", RandString(16), procVer)
		if err != nil {
			t.Fatal(err)
		}

		vec := testlib.RandNormalizedEmbedding(dimensions)
		err = s.PutEmbedding(t.Context(), sourceUUID, file.UUID, "chunk1", vec)
		if err != nil {
			t.Fatal(err)
		}

		// Delete source
		err = s.DeleteSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		// Search in that source should return empty or error
		results, err := s.SearchSimilarEmbedddings(t.Context(), vec, []storage.SourceUUID{sourceUUID}, 10)
		if err != nil {
			// Either error or empty results is acceptable depending on impl
			t.Logf("expected error or empty after deletion, got error: %v", err)
		} else if len(results) != 0 {
			t.Errorf("expected 0 results after source deletion, got %d", len(results))
		}
	})

	t.Run("PutEmbeddingInvalidVector", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}
		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "invalid-input-model"}
		file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, "/file/invalid.txt", RandString(16), procVer)
		if err != nil {
			t.Fatal(err)
		}

		invalidVectors := [][]float32{
			nil,
			{},
		}

		for i, vec := range invalidVectors {
			err := s.PutEmbedding(t.Context(), sourceUUID, file.UUID, "chunk-invalid", vec)
			if err == nil {
				t.Errorf("expected error for invalid vector #%d but got none", i)
			}
		}
	})

	t.Run("SearchLimitRespected", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}
		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "limit-test-model"}
		file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, "/file/limit.txt", RandString(16), procVer)
		if err != nil {
			t.Fatal(err)
		}

		// Put 20 embeddings
		for i := 0; i < 20; i++ {
			vec := testlib.RandNormalizedEmbedding(dimensions)
			err := s.PutEmbedding(t.Context(), sourceUUID, file.UUID, "chunk-"+RandString(5), vec)
			if err != nil {
				t.Fatal(err)
			}
		}

		// Search with limit 5
		results, err := s.SearchSimilarEmbedddings(t.Context(), testlib.RandNormalizedEmbedding(dimensions), []storage.SourceUUID{sourceUUID}, 5)
		if err != nil {
			t.Fatal(err)
		}

		if len(results) > 5 {
			t.Errorf("expected max 5 results, got %d", len(results))
		}
	})

	t.Run("DeleteFileDeletesEmbeddings", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		procVer := storage.ProcessorVersion{Major: 1, Minor: 0, Patch: 0, EmbeddingsModel: "delete-file-embedding"}
		file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, "/file/delete-embedding.txt", RandString(16), procVer)
		if err != nil {
			t.Fatal(err)
		}

		// Put several embeddings for this file
		for i := 0; i < 5; i++ {
			vec := testlib.RandNormalizedEmbedding(dimensions)
			chunk := "chunk-" + RandString(5)
			err := s.PutEmbedding(t.Context(), sourceUUID, file.UUID, chunk, vec)
			if err != nil {
				t.Fatal(err)
			}
		}

		// Pick one embedding vector to test search
		testVector := testlib.RandNormalizedEmbedding(dimensions)
		err = s.PutEmbedding(t.Context(), sourceUUID, file.UUID, "test-chunk", testVector)
		if err != nil {
			t.Fatal(err)
		}

		// Confirm search finds embedding before deletion
		results, err := s.SearchSimilarEmbedddings(t.Context(), testVector, []storage.SourceUUID{sourceUUID}, 10)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, emb := range results {
			if emb.File.UUID == file.UUID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find embedding before file deletion, but did not")
		}

		// Delete file
		err = s.DeleteFile(t.Context(), sourceUUID, file.UUID)
		if err != nil {
			t.Fatal(err)
		}

		// Search again, should NOT find any embedding for deleted file
		resultsAfterDelete, err := s.SearchSimilarEmbedddings(t.Context(), testVector, []storage.SourceUUID{sourceUUID}, 10)
		if err != nil {
			t.Fatal(err)
		}
		for _, emb := range resultsAfterDelete {
			if emb.File.UUID == file.UUID {
				t.Errorf("found embedding for deleted file %s after deletion", file.UUID)
			}
		}
	})

	t.Run("PutEmbeddingOnNonexistentFile", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		nonExistentFileUUID := storage.FileUUID(RandString(16))
		vec := testlib.RandNormalizedEmbedding(dimensions)

		err = s.PutEmbedding(t.Context(), sourceUUID, nonExistentFileUUID, "chunk", vec)
		if err == nil {
			t.Error("expected error when putting embedding on nonexistent file, got nil")
		}
	})

	t.Run("SearchWithEmptyVector", func(t *testing.T) {
		emptyVec := []float32{}

		_, err := s.SearchSimilarEmbedddings(t.Context(), emptyVec, []storage.SourceUUID{}, 10)
		if err == nil {
			t.Error("expected error when searching with empty vector, got nil")
		}
	})

	t.Run("ConcurrentPutAndSearch", func(t *testing.T) {
		sourceUUID := storage.SourceUUID(RandString(32))
		_, err := s.GetOrCreateSource(t.Context(), sourceUUID)
		if err != nil {
			t.Fatal(err)
		}

		file, _, err := s.GetOrCreateFile(t.Context(), sourceUUID, "/concurrent/file.txt", RandString(16), storage.ProcessorVersion{Major: 1})
		if err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		errs := make(chan error, 100)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				vector := testlib.RandNormalizedEmbedding(dimensions)
				chunk := "chunk-" + RandString(5)
				if err := s.PutEmbedding(t.Context(), sourceUUID, file.UUID, chunk, vector); err != nil {
					errs <- err
					return
				}
			}
		}()

		// Run searches concurrently with puts
		for i := 0; i < 10; i++ {
			vector := testlib.RandNormalizedEmbedding(dimensions)
			results, err := s.SearchSimilarEmbedddings(t.Context(), vector, []storage.SourceUUID{sourceUUID}, 5)
			if err != nil {
				t.Fatal(err)
			}
			// Optionally check results length or properties
			_ = results
		}

		wg.Wait()
		close(errs)

		for e := range errs {
			if e != nil {
				t.Fatal(e)
			}
		}
	})
}
