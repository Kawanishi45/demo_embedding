package controller

import (
  "github.com/Kawanishi45/demo_embedding/cron"
  "github.com/gin-gonic/gin"
  "github.com/jmoiron/sqlx"
  "net/http"
)

type Question struct {
  Text string `json:"text"`
}

func (s *Server) AskQuestionHandler(c *gin.Context) {
  var q Question
  if err := c.BindJSON(&q); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  queryVector, err := cron.GetEmbeddingVector(q.Text)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  rows, err := s.DB.Queryx(`
        SELECT document_id, chunk_index, chunk_text
        FROM embeddings
        ORDER BY embedding <=> $1
        LIMIT 3
    `, queryVector)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  var chunks []struct {
    DocumentID int    `db:"document_id"`
    ChunkIndex int    `db:"chunk_index"`
    ChunkText  string `db:"chunk_text"`
  }
  for rows.Next() {
    var chunk struct {
      DocumentID int    `db:"document_id"`
      ChunkIndex int    `db:"chunk_index"`
      ChunkText  string `db:"chunk_text"`
    }
    if err := rows.StructScan(&chunk); err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      return
    }
    chunks = append(chunks, chunk)
  }

  contextText := ""
  for _, chunk := range chunks {
    var contextRows *sqlx.Rows
    contextRows, err = s.DB.Queryx(`
            SELECT chunk_text
            FROM embeddings
            WHERE document_id = $1
            AND chunk_index BETWEEN $2 - 1 AND $2 + 1
        `, chunk.DocumentID, chunk.ChunkIndex)
    if err != nil {
      c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
      return
    }

    for contextRows.Next() {
      var contextChunk struct {
        ChunkText string `db:"chunk_text"`
      }
      if err := contextRows.StructScan(&contextChunk); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
      }
      contextText += contextChunk.ChunkText + "\n"
    }
  }

  response, err := getAIResponse(contextText, q.Text)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "prompt":   contextText + "\n" + q.Text,
    "response": response,
  })
}

func getAIResponse(context, question string) (string, error) {
  // OpenAIのAPIを使用して質問に対する回答を取得するコードを実装
  // ここでは仮のコードを使用
  return "これは仮の回答です。", nil
}
