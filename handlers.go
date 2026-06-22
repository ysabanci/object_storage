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
)

func addObject(w http.ResponseWriter, r *http.Request, db *sql.DB) { //dosya ekleme handleri

	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	fullPath, err := pathControl(bucket, key)
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

	//header'dan content type cektigimiz kod blogu burasi
	contentType := r.Header.Get("Content-Type") // varsayılan curl değerini de bos kabul et
	if contentType == "" || contentType == "application/x-www-form-urlencoded" {
		contentType = mime.TypeByExtension(filepath.Ext(key))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	//on conflict ile eger dosya var ise ustune yaz/guncelle mekanizmasini ekledik.
	query := `
    INSERT INTO objects (bucket, key, size, content_type)
    VALUES ($1, $2, $3, $4)
    ON CONFLICT (bucket, key) DO UPDATE
    SET size = $3, content_type = $4, created_at = NOW()
`
	_, err = db.Exec(query, bucket, key, size, contentType)
	if err != nil {
		_ = os.Remove(fullPath)
		http.Error(w, "Dosya diske yazıldı fakat metadata veritabanina eklenemedi", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, "Yaratma işlemi gerçekleştirildi.")
}

// addObject fonksiyonumuz zaten MkdirAll sayesinde dosyaya ait tum pathi olusturur. bos bir kova olusturmak istenirse
// diye bu fonksiyonu yazdim.
func addBucket(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	folderPath, err := pathControl(bucket, "")
	if err != nil {
		http.Error(w, "Geçersiz kova adı", http.StatusBadRequest)
		return
	}
	err = os.MkdirAll(folderPath, 0775)
	if err != nil {
		http.Error(w, "Kova olusturulamadi", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func readObject(w http.ResponseWriter, r *http.Request) {
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

	fullPath, err := pathControl(bucket, key)
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

func listBuckets(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.Query(`SELECT DISTINCT bucket FROM objects`)
	if err != nil {
		http.Error(w, "Bucketlar listelenemedi.", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var buckets []string
	for rows.Next() {
		var bucket string
		err := rows.Scan(&bucket)
		if err != nil {
			log.Println("Satir okunamadi:", err)
			continue
		}
		buckets = append(buckets, bucket)
	}
	w.Header().Set("Content-Type", "text/plain")
	for _, value := range buckets {
		_, _ = fmt.Fprintln(w, value)
	}
}

func listObjects(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucket := r.PathValue("bucket")
	rows, err := db.Query(`SELECT key, size, content_type, created_at FROM objects WHERE bucket = $1`, bucket)
	if err != nil {
		http.Error(w, "Objectler listelenemedi.", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/plain")

	for rows.Next() {
		var key string
		var size int64
		var contentType string
		var createdAt time.Time
		err := rows.Scan(&key, &size, &contentType, &createdAt)
		if err != nil {
			log.Println("Dosya okunamadi:", err)
			continue
		}
		_, _ = fmt.Fprintf(w, "%s | %d byte | %s | %s\n", key, size, contentType, createdAt.Format("2006-01-02 15:04"))
	}
}

// guvenlik amacli path kontrolu, kod tekrari olmamasi icin fonksiyon olarak yazildi.
func pathControl(kovaAdi, dosyaYolu string) (string, error) {
	fullPath := filepath.Join(baseStorageDir, kovaAdi, dosyaYolu)
	if !strings.HasPrefix(fullPath, filepath.Clean(baseStorageDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("geçersiz yol:")
	}
	return fullPath, nil
}
