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

func garbageCollector(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		log.Println("hatali kayit araniyor.")

		rows, err := db.Query(`SELECT bucket, key FROM objects`)

		if err != nil {
			log.Println("Veritabanina baglanamadi", err)
			continue // dongunun basina don
		}

		for rows.Next() {
			var bucket, key string
			err = rows.Scan(&bucket, &key)
			if err != nil {
				log.Println("Satir okunamadi.", err)
				continue
			}
			fullPath := filepath.Join(baseStorageDir, bucket, key)
			_, err := os.Stat(fullPath)
			if os.IsNotExist(err) {
				log.Printf("Bulundu: %s/%s. siliniyor...\n", bucket, key)

				_, deleteErr := db.Exec(`DELETE FROM objects WHERE bucket = $1 AND key = $2`, bucket, key)
				if deleteErr != nil {
					log.Println("Silinemedi:", deleteErr)
				}
			}
		}
		rows.Close()
	}
}

func createTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS objects (
		id SERIAL PRIMARY KEY,
		bucket VARCHAR(255) NOT NULL,
		key VARCHAR(1024) NOT NULL,
		size BIGINT NOT NULL,
		content_type VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(bucket, key)
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Tablolar olusturulurken hata cikti: ", err)
	}
}
