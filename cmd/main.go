package main

import (
  "github.com/Kawanishi45/demo_embedding/controller"
  "github.com/Kawanishi45/demo_embedding/cron"
  "github.com/gin-contrib/cors"
  "github.com/gin-gonic/gin"
  "github.com/jmoiron/sqlx"
  _ "github.com/lib/pq"
  "log"
  "time"
)

var db *sqlx.DB

func initDB() {
  var err error
  db, err = sqlx.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
  if err != nil {
    log.Fatalln(err)
  }
}

func main() {
  initDB()
  router := gin.Default()

  // CORSミドルウェアの設定
  router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"*"}, // 必要に応じてドメインを変更
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
  }))

  ControllerServer := controller.Server{DB: db}
  router.POST("/add_document", ControllerServer.AddDocumentHandler)
  router.POST("/ask_question", ControllerServer.AskQuestionHandler)

  CronServer := cron.Server{DB: db}
  go CronServer.VectorizeChunks()

  router.Run(":8080")
}
