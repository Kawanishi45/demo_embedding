package cron

import (
  "context"
  "fmt"
  "github.com/Kawanishi45/demo_embedding/helper"
  "github.com/sashabaranov/go-openai"
  "log"
  "os"
  "time"
)

func (s *Server) VectorizeChunks() {
  for {
    rows, err := s.DB.Queryx("SELECT id, chunk_text FROM embeddings WHERE embedding IS NULL")
    if err != nil {
      log.Println("Error fetching unvectorized chunks:", err)
      continue
    }

    var chunks []struct {
      ID        int    `db:"id"`
      ChunkText string `db:"chunk_text"`
    }
    for rows.Next() {
      var chunk struct {
        ID        int    `db:"id"`
        ChunkText string `db:"chunk_text"`
      }
      if err = rows.StructScan(&chunk); err != nil {
        log.Println("Error scanning chunk:", err)
        continue
      }
      chunks = append(chunks, chunk)
    }

    for _, chunk := range chunks {
      var vector []float32
      vector, err = GetEmbeddingVector(chunk.ChunkText)
      if err != nil {
        log.Println("Error getting embedding vector:", err)
        continue
      }

      _, err = s.DB.Exec("UPDATE embeddings SET embedding = $1 WHERE id = $2", helper.Vector(vector), chunk.ID)
      if err != nil {
        log.Println("Error updating embedding:", err)
        continue
      }
    }

    time.Sleep(20 * time.Second)
  }
}

// GetEmbeddingVector テキストをベクトル化し、JSON形式で返す関数
func GetEmbeddingVector(text string) ([]float32, error) {
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    return nil, fmt.Errorf("OPENAI_API_KEY is not set")
  }

  client := openai.NewClient(apiKey)

  req := openai.EmbeddingRequest{
    Model: openai.AdaEmbeddingV2, // 適切なモデルを選択
    Input: []string{text},
  }

  resp, err := client.CreateEmbeddings(context.Background(), req)
  if err != nil {
    return nil, err
  }

  if len(resp.Data) == 0 {
    return nil, fmt.Errorf("no embedding response from OpenAI")
  }

  embedding := resp.Data[0].Embedding
  return embedding, nil
}
