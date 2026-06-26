package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func initDB() *sql.DB {
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	if dbPass == "" {
		log.Fatal("Veritabani sifresi tanimlanmamis")
	}
	connStr := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s", dbUser, dbPass, dbName) //guvenlik amacli .env olarak
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal("Veritabani baslatilamadi: ", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Veritabanina ulasilamiyor.", err)
	}
	fmt.Println("Postgres baglantisi basarili")
	return db
}

type ObjectTarget struct {
	Bucket string
	Key    string
}

func garbageCollector(db *sql.DB) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		log.Println("hatali kayit araniyor.")
		saveAndclean(db)
	}
}
func saveAndclean(db *sql.DB) {
	var targets []ObjectTarget

	rows, err := db.Query(`SELECT bucket, key FROM objects`)

	if err != nil {
		log.Println("Veritabanina baglanamadi", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var bucket, key string
		err = rows.Scan(&bucket, &key)
		if err != nil {
			log.Println("Satir okunamadi.", err)
			continue
		}
		targets = append(targets, ObjectTarget{Bucket: bucket, Key: key})
	}
	if err := rows.Err(); err != nil {
		log.Println("Döngü sırasında hata oluştu:", err)
	}

	for _, t := range targets {
		fullPath := filepath.Join(baseStorageDir, t.Bucket, t.Key)
		_, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			log.Printf("Bulundu: %s/%s. siliniyor...\n", t.Bucket, t.Key)

			_, deleteErr := db.Exec(`DELETE FROM objects WHERE bucket = $1 AND key = $2`, t.Bucket, t.Key)
			if deleteErr != nil {
				log.Println("Silinemedi:", deleteErr)
			}
		}
	}

}
func createTable(db *sql.DB) {
	BucketQuery := `
	CREATE TABLE IF NOT EXISTS buckets (
		name VARCHAR(255) PRIMARY KEY,
	    is_public BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT NOW()
	);`
	_, err := db.Exec(BucketQuery)
	if err != nil {
		log.Fatal("Bucket tablosu olusturulurken hata cikti: ", err)
	}

	ObjectQuery := `
	CREATE TABLE IF NOT EXISTS objects (
		id SERIAL PRIMARY KEY,
		bucket VARCHAR(255) NOT NULL REFERENCES buckets(name) ON DELETE CASCADE,
		key VARCHAR(1024) NOT NULL,
		size BIGINT NOT NULL,
		content_type VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(bucket, key)
	);`

	_, err = db.Exec(ObjectQuery)
	if err != nil {
		log.Fatal("Object tablosu olusturulurken hata cikti: ", err)
	}
}
