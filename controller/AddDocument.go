package controller

import (
  "github.com/gin-gonic/gin"
  "log"
  "net/http"
  "strings"
)

type Document struct {
  Title   string `json:"title"`
  Author  string `json:"author"`
  Content string `json:"content"`
}

func (s *Server) AddDocumentHandler(c *gin.Context) {
  var doc Document
  if err := c.BindJSON(&doc); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  var documentID int
  err := s.DB.QueryRow("INSERT INTO documents (title, author, content) VALUES ($1, $2, $3) RETURNING id", doc.Title, doc.Author, doc.Content).Scan(&documentID)
  if err != nil {
    log.Printf("Error inserting document: %v\n", err)
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting document"})
    return
  }

  chunks := splitContentIntoChunks(doc.Content)

  for i, chunk := range chunks {
    _, err = s.DB.Exec("INSERT INTO embeddings (document_id, embedding, chunk_index, chunk_text) VALUES ($1, null, $2, $3)", documentID, i, chunk)
    if err != nil {
      log.Printf("Error inserting embedding: %v\n", err)
      c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting embedding"})
      return
    }
  }

  c.JSON(http.StatusOK, gin.H{"status": "document added"})
}

func splitContentIntoChunks(content string) []string {
  // contentを一定の閾値（例えば1000文字）以上の長さの句点で区切る
  sentences := strings.Split(content, "。")
  var chunks []string
  var chunk string

  for _, sentence := range sentences {
    if len(chunk)+len(sentence) > 1000 {
      chunks = append(chunks, chunk)
      chunk = sentence + "。"
    } else {
      chunk += sentence + "。"
    }
  }
  if len(chunk) > 0 {
    chunks = append(chunks, chunk)
  }

  return chunks
}
