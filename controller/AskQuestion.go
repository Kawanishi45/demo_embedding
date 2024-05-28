package controller

import (
  "context"
  "fmt"
  "github.com/Kawanishi45/demo_embedding/cron"
  "github.com/Kawanishi45/demo_embedding/helper"
  "github.com/gin-gonic/gin"
  "github.com/sashabaranov/go-openai"
  "log"
  "math"
  "net/http"
  "os"
  "sort"
)

type question struct {
  Text string `json:"text"`
}

type roughResult struct {
  DocumentID int `db:"document_id"`
  ChunkIndex int `db:"chunk_index"`
}

type preciseResult struct {
  DocumentID int
  ChunkIndex int
  Distance   float64
}

func (s *Server) AskQuestionHandler(c *gin.Context) {
  var q question
  if err := c.BindJSON(&q); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  queryVector, err := cron.GetEmbeddingVector(q.Text)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  // まずは粗い近似検索
  roughResults, err := s.roughApproximateSearch(queryVector)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  // 粗い検索結果に基づいて厳密な検索を実行
  preciseResults, err := s.preciseSearch(queryVector, roughResults)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  // 上位3つの結果を取得
  topChunks := getTopChunks(preciseResults)

  contextText := s.getContextText(topChunks)

  response, err := s.getAIResponse(contextText, q.Text)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "prompt":   contextText + "\n" + q.Text,
    "response": response,
  })
}

func (s *Server) roughApproximateSearch(queryVector []float32) ([]roughResult, error) {
  rows, err := s.DB.Queryx(`
        SELECT document_id, chunk_index
        FROM embeddings
        ORDER BY embedding <=> $1
        LIMIT 100 -- 粗い検索ではトップ100件を取得
    `, helper.Vector(queryVector))
  if err != nil {
    return nil, err
  }

  var results []roughResult
  for rows.Next() {
    var result roughResult
    if err = rows.StructScan(&result); err != nil {
      return nil, err
    }
    results = append(results, result)
  }
  return results, nil
}

func (s *Server) preciseSearch(queryVector []float32, roughResults []roughResult) ([]preciseResult, error) {
  var results []preciseResult

  for _, result := range roughResults {
    var embeddingStr string
    err := s.DB.Get(&embeddingStr, "SELECT embedding FROM embeddings WHERE document_id = $1 AND chunk_index = $2", result.DocumentID, result.ChunkIndex)
    if err != nil {
      return nil, err
    }

    embedding, err := helper.StringToVector(embeddingStr)
    if err != nil {
      return nil, err
    }

    distance := cosineDistance(queryVector, embedding)
    results = append(results, struct {
      DocumentID int
      ChunkIndex int
      Distance   float64
    }{
      DocumentID: result.DocumentID,
      ChunkIndex: result.ChunkIndex,
      Distance:   distance,
    })
  }

  // 距離でソート
  sort.Slice(results, func(i, j int) bool {
    return results[i].Distance < results[j].Distance
  })

  return results, nil
}

func cosineDistance(a []float32, b []float64) float64 {
  var dotProduct, normA, normB float64
  for i := range a {
    dotProduct += float64(a[i]) * b[i]
    normA += float64(a[i]) * float64(a[i])
    normB += b[i] * b[i]
  }
  return 1.0 - (dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func getTopChunks(results []preciseResult) []struct{ DocumentID, ChunkIndex int } {
  var topChunks []struct {
    DocumentID int
    ChunkIndex int
  }

  for i := 0; i < 3 && i < len(results); i++ {
    topChunks = append(topChunks, struct {
      DocumentID int
      ChunkIndex int
    }{
      DocumentID: results[i].DocumentID,
      ChunkIndex: results[i].ChunkIndex,
    })
  }
  return topChunks
}

func (s *Server) getContextText(chunks []struct{ DocumentID, ChunkIndex int }) string {
  contextText := ""
  for _, chunk := range chunks {
    contextRows, err := s.DB.Queryx(`
            SELECT chunk_text
            FROM embeddings
            WHERE document_id = $1
            AND chunk_index BETWEEN $2 - 1 AND $2 + 1
        `, chunk.DocumentID, chunk.ChunkIndex)
    if err != nil {
      log.Println("Error fetching context text:", err)
      continue
    }

    for contextRows.Next() {
      var contextChunk struct {
        ChunkText string `db:"chunk_text"`
      }
      if err := contextRows.StructScan(&contextChunk); err != nil {
        log.Println("Error scanning context chunk:", err)
        continue
      }
      contextText += contextChunk.ChunkText + "\n"
    }
  }
  return contextText
}

func (s *Server) getAIResponse(contextText, question string) (string, error) {
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    return "", fmt.Errorf("OPENAI_API_KEY is not set")
  }

  client := openai.NewClient(apiKey)
  messages := []openai.ChatCompletionMessage{
    {
      Role:    "system",
      Content: "You are a helpful assistant.",
    },
    {
      Role:    "user",
      Content: contextText,
    },
    {
      Role:    "user",
      Content: question,
    },
  }

  req := openai.ChatCompletionRequest{
    Model:       openai.GPT3Dot5Turbo,
    Messages:    messages,
    Temperature: 0.1,
    TopP:        0.1,
    //MaxTokens:   150,
  }

  resp, err := client.CreateChatCompletion(context.Background(), req)
  if err != nil {
    return "", err
  }

  if len(resp.Choices) == 0 {
    return "", fmt.Errorf("no response from OpenAI")
  }

  _, err = s.DB.Exec("INSERT INTO questions (question, context, response) VALUES ($1, $2, $3)", question, contextText, resp.Choices[0].Message.Content)
  if err != nil {
    log.Println("Error inserting question:", err)
  }

  return resp.Choices[0].Message.Content, nil
}
