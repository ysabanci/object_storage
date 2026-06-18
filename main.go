package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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
	for {
		time.Sleep(1 * time.Hour)
		log.Println("hatali kayit araniyor.")

		rows, err := db.Query(`SELECT bucket, key FROM objects`)

		if err != nil {
			log.Println("Veritabanina baglanamadi", err)
			continue // dongunun basina don
		}
		for rows.Next() {
			var bucket, key string
			rows.Scan(&bucket, &key)
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

const baseStorageDir = "/Users/ysabanci/Desktop/s3" //sabit dizin

func addObject(w http.ResponseWriter, r *http.Request, db *sql.DB) { //dosya ekleme handleri

	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	fullPath, err := pathControl(r.PathValue("bucket"), r.PathValue("key"))
	if err != nil {
		http.Error(w, "Geçersiz yol", http.StatusBadRequest)
		return
	}
	folderPath := filepath.Dir(fullPath)

	err = os.MkdirAll(folderPath, 0775) //folderPathi tum yolu baz alarak komple olusturur.
	if err != nil {
		http.Error(w, "Klasör oluşturma hatası", http.StatusInternalServerError)
		return
	}

	newFile, err := os.Create(fullPath)
	if err != nil {
		http.Error(w, "Dosya oluşturma hatası", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	size, err := io.Copy(newFile, r.Body)
	if err != nil {
		http.Error(w, "Dosya yükleme hatası", http.StatusInternalServerError)
		return
	} //exception handling

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(key)) // dosya uzantisina gore uzanti belirleyip kategorize ettigimiz kisim
		if contentType == "" {
			contentType = "application/octet-stream" // bilinmeyen dosyalari kayit ederiz.
		}
	} //header'dan content type cektigimiz kod blogu burasi

	query := `
    INSERT INTO objects (bucket, key, size, content_type)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (bucket, key) DO UPDATE
    SET size = $3, content_type = $4, created_at = NOW()
` //on conflict ile eger dosya var ise ustune yaz/guncelle mekanizmasini ekledik.
	_, err = db.Exec(query, bucket, key, size, contentType)
	if err != nil {
		os.Remove(fullPath)
		http.Error(w, "Dosya diske yazıldı fakat metadata veritabanina eklenemedi", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Yaratma işlemi gerçekleştirildi.")
}

func readObject(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	fullPath, err := pathControl(r.PathValue("bucket"), r.PathValue("key"))
	if err != nil {
		http.Error(w, "Geçersiz yol", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, fullPath)

}

func deleteObject(w http.ResponseWriter, r *http.Request, db *sql.DB) { //dosya silen fonksiyon
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	fullPath, err := pathControl(r.PathValue("bucket"), r.PathValue("key"))
	if err != nil {
		http.Error(w, "Geçersiz yol", http.StatusBadRequest)
		return
	}
	err = os.Remove(fullPath)
	if err != nil {
		http.Error(w, "Dosya silinemedi veya bulunamadı", http.StatusNotFound)
		return
	}

	query := `DELETE FROM objects WHERE bucket = $1 AND key = $2`
	_, err = db.Exec(query, bucket, key)
	if err != nil {
		log.Println("Veritabanindan silinemedi:", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathControl(kovaAdi, dosyaYolu string) (string, error) { // guvenlik amacli path kontrolu, kod tekrari olmamasi icin fonksiyon olarak yazildi.
	fullPath := filepath.Join(baseStorageDir, kovaAdi, dosyaYolu)
	if !strings.HasPrefix(fullPath, filepath.Clean(baseStorageDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("geçersiz yol:")
	}
	return fullPath, nil
}

func main() {
	db := initDB()          //db initialiton
	defer db.Close()        //main kapaninca dosyayi kapatir
	createTable(db)         //tablo olusturma
	go garbageCollector(db) //temizlikci goroutine

	mux := http.NewServeMux() //deafult olursa guvenlik sorunu olabilirdi, mux handle'i yazildi

	mux.HandleFunc("PUT /buckets/{bucket}/objects/{key...}", func(w http.ResponseWriter, r *http.Request) {
		addObject(w, r, db)
	})

	mux.HandleFunc("GET /buckets/{bucket}/objects/{key...}", func(w http.ResponseWriter, r *http.Request) {
		readObject(w, r, db)
	})

	mux.HandleFunc("DELETE /buckets/{bucket}/objects/{key...}", func(w http.ResponseWriter, r *http.Request) {
		deleteObject(w, r, db)
	}) // dependency injection, cunku mux 2 parametre bekliyordu ve db ekleyemiyordum. bundan sonra db parametresine de uygun sekilde calisir.

	fmt.Println("Sistem 8080 portunda çalışıyor...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
