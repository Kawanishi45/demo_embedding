package cron

import (
  "encoding/json"
  "log"
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
      if err := rows.StructScan(&chunk); err != nil {
        log.Println("Error scanning chunk:", err)
        continue
      }
      chunks = append(chunks, chunk)
    }

    for _, chunk := range chunks {
      var vector string
      vector, err = GetEmbeddingVector(chunk.ChunkText)
      if err != nil {
        log.Println("Error getting embedding vector:", err)
        continue
      }

      _, err = s.DB.Exec("UPDATE embeddings SET embedding = $1 WHERE id = $2", vector, chunk.ID)
      if err != nil {
        log.Println("Error updating embedding:", err)
        continue
      }
    }

    time.Sleep(20 * time.Second)
  }
}

// GetEmbeddingVector テキストをベクトル化し、JSON形式で返す関数
func GetEmbeddingVector(text string) (string, error) {
  // OpenAIのAPIを使用してtextをベクトル化するコードを実装
  // ここでは仮のコードを使用
  vector := make([]float32, 768) // 例: 768次元のベクトル

  // 例として、ベクトルの一部を埋めます（実際にはAPIから取得）
  for i := range vector {
    vector[i] = float32(i)
  }

  // []float32をJSON文字列に変換
  vectorJSON, err := json.Marshal(vector)
  if err != nil {
    return "", err
  }
  return string(vectorJSON), nil
}
