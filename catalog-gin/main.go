package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	ID int64 `json:"id"`
	Name string `json:"name" binding:"required"`
	Description string `json:"description"`
	Score float32 `json:"score,omitempty"`
}

type SearchQuery struct {
	Q string `form:"q" binding:"required"`
	Mode string `form:"mode" binding:"omitempty,oneof=fts fts_norank"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

func tsQuery(q string) string {
	return strings.Join(wordRe.FindAllString(strings.ToLower(q), -1), " | ")
}

func main() {
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/shop"))
	if err != nil {
		log.Fatal(err)
	}
	cfg.MaxConns = 20

	var pool *pgxpool.Pool
	for i := 0; ; i++ {
		if pool, err := pgxpool.NewWithConfig(ctx, cfg); err == nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
		}
		if i >= 30 {
			log.Fatalf("tidak bisa konek database: %v", err)
		}
		time.Sleep(2 * time.Second)
	}
	defer pool.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok - 200"})
	})

	r.GET("/products/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id harus angka"})
			return
		}
		var p Product
		err = pool.QueryRow(c.Request.Context(),
			`SELECT id, name, description FROM products WHERE id = $1`, id).
			Scan(&p.ID, &p.Name, &p.Description)
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	})

	r.GET("/products/search", func(c *gin.Context) {
		var sq SearchQuery
		if err := c.ShouldBindQuery(&sq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if sq.Mode == "" {
			sq.Mode = "fts"
		}
		if sq.Limit == 0 {
			sq.Limit = 10
		}
		tq := tsQuery(sq.Q)
		if tq == "" {
			c.JSON(http.StatusOK, gin.H{"total": 0, "hits": []Product{}})
			return
		}

		sql := `SELECT id, name, description, 0::float4
		        FROM products, to_tsquery('simple', $1) AS query
		        WHERE tsv @@ query LIMIT $2`
		if sq.Mode == "fts" {
			sql = `SELECT id, name, description, ts_rank(tsv, query)
			       FROM products, to_tsquery('simple', $1) AS query
			       WHERE tsv @@ query ORDER BY 4 DESC, id LIMIT $2`
		}
		rows, err := pool.Query(c.Request.Context(), sql, tq, sq.Limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		hits := []Product{}
		for rows.Next() {
			var p Product
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Score); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			hits = append(hits, p)
		}
		c.JSON(http.StatusOK, gin.H{"total": len(hits), "hits": hits})
	})

	r.POST("/products", func(c *gin.Context) {
		var p Product
		if err := c.ShouldBindJSON(&p); err != nil || strings.TrimSpace(p.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name wajib diisi"})
			return
		}
		err := pool.QueryRow(c.Request.Context(),
			`INSERT INTO products (name, description) VALUES ($1, $2) RETURNING id`,
			p.Name, p.Description).Scan(&p.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, p)
	})

	port := getenv("PORT", "8091")
	log.Println("catalog-gin listening on :" + port)
	log.Fatal(r.Run(":" + port))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}