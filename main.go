package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var db *sql.DB

//go:embed tracker.js
var trackerJS embed.FS

type TrackingData struct {
	UserID    string    `json:"userId"`
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	var err error
	db, err = sql.Open("sqlite", "./tracking.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable()

	r := gin.Default()

	// CORS設定
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}
	r.Use(cors.New(config))

	r.Use(gin.Recovery())

	r.POST("/track", trackHandler)
	r.GET("/tracker.js", trackerJSHandler)
	r.GET("/generate-user-id", generateUserIDHandler)
	r.GET("/events", getEventsHandler)

	log.Println("Server is running on http://localhost:8080")
	r.Run(":8080")
}

func createTable() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT,
			event TEXT,
			timestamp DATETIME
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func trackHandler(c *gin.Context) {
	var data TrackingData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("INSERT INTO events (user_id, event, timestamp) VALUES (?, ?, ?)",
		data.UserID, data.Event, data.Timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func generateUserIDHandler(c *gin.Context) {
	userID := uuid.New().String()
	c.JSON(http.StatusOK, gin.H{
		"userId": userID,
	})
}

func trackerJSHandler(c *gin.Context) {
	c.Header("Content-Type", "application/javascript")

	// embed.FSからサブファイルシステムを取得
	fsys, err := fs.Sub(trackerJS, ".")
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not access tracker.js")
		return
	}

	// ファイルを直接提供
	c.FileFromFS("tracker.js", http.FS(fsys))
}

type Event struct {
	ID        int       `json:"id"`
	UserID    string    `json:"userId"`
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
}

func getEventsHandler(c *gin.Context) {
	limit := 100 // デフォルトの制限
	offset := 0

	// クエリパラメータから limit と offset を取得
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil {
			offset = parsedOffset
		}
	}

	// イベントを取得
	query := `SELECT id, user_id, event, timestamp FROM events ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.ID, &event.UserID, &event.Event, &event.Timestamp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan event"})
			return
		}
		events = append(events, event)
	}

	// 総イベント数を取得
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"meta": gin.H{
			"total":  totalCount,
			"limit":  limit,
			"offset": offset,
		},
	})
}
