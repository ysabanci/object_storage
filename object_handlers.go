package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func addObject(w http.ResponseWriter, r *http.Request, db *sql.DB) { //dosya ekleme handleri

	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	fullPath, err := pathControl(bucket, key)
	if err != nil {
		sendJSONresponse(w, 400, "Error", "Invalid path")
		return
	}
	folderPath := filepath.Dir(fullPath)

	var exist bool

	err = db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM buckets WHERE name = $1)`, bucket).Scan(&exist)

	if err != nil {
		log.Println("Bucket sorgusu hatasi:", err)
		sendJSONresponse(w, 500, "Error", "Database error")
		return
	}

	if !exist {
		sendJSONresponse(w, 404, "Error", "Bucket does not exist, bucket must be created first")
		return
	}

	err = os.MkdirAll(folderPath, 0775) //folderPathi tum yolu baz alarak komple olusturur.
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Failed to create directory")
		return
	}

	newFile, err := os.Create(fullPath)
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Failed to create file")
		return
	}
	defer newFile.Close()

	size, err := io.Copy(newFile, r.Body)
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Failed to upload file")
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
		newFile.Close()
		deleteErr := os.Remove(fullPath)
		if deleteErr != nil {
			log.Printf("DİKKAT: Veritabanı hatası sonrası çöp dosya silinemedi: %s", fullPath)
		}
		sendJSONresponse(w, 500, "Error", "File written to disk but metadata could not be saved to database")
	}

	sendJSONresponse(w, 201, "Success", "File uploaded successfully")
}

func readObject(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var isPublic bool //once kimlik dogrulanir

	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	err := db.QueryRow("SELECT is_public FROM buckets WHERE name = $1", bucket).Scan(&isPublic)
	if err != nil {
		// Kova bulunamadıysa veya hata varsa güvenlik gereği "false" kabul et.
		isPublic = false
	}

	if !isPublic {
		apiKey := os.Getenv("API_KEY")
		clientKey := r.Header.Get("X-API-Key")
		isAuthorized := clientKey == apiKey

		if !isAuthorized { //presigned url kontrolu
			expires := r.URL.Query().Get("expires")
			signature := r.URL.Query().Get("signature")

			if expires != "" && signature != "" {
				expiresInt, err := strconv.ParseInt(expires, 10, 64)
				if err != nil {
					sendJSONresponse(w, 400, "Error", "Invalid time format. Please use a valid format")
					return
				}
				if time.Now().Unix() > expiresInt {
					sendJSONresponse(w, 401, "Error", "Link has expired")
					return
				}
				expectedSignature := createSignature(bucket, key, expires, apiKey)
				if signature == expectedSignature {
					isAuthorized = true
				}
			}
		}
		if !isAuthorized { //erisim reddi buradadir
			sendJSONresponse(w, 401, "Error", "Access denied: Invalid credentials")
			return
		}
	}

	fullPath, err := pathControl(r.PathValue("bucket"), r.PathValue("key"))
	if err != nil {
		sendJSONresponse(w, 400, "Error", "Invalid path")
		return
	}
	var contentType string //sonra varlik dogrulanir ve type cekilir

	query := `SELECT content_type FROM objects WHERE bucket = $1 AND key = $2`
	err = db.QueryRow(query, bucket, key).Scan(&contentType)
	if err == sql.ErrNoRows {
		sendJSONresponse(w, 404, "Error", "Object does not exist")
		return
	} else if err != nil {
		log.Println("Veritabanindan okunamadi:", err)
		sendJSONresponse(w, 500, "Error", "Database connection error")
		return
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, fullPath)

}

func deleteObject(w http.ResponseWriter, r *http.Request, db *sql.DB) { //dosya silen fonksiyon
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")
	fullPath, err := pathControl(bucket, key)
	if err != nil {
		sendJSONresponse(w, 400, "Error", "Invalid bucket path")
		return
	}

	query := `DELETE FROM objects WHERE bucket = $1 AND key = $2`
	result, err := db.Exec(query, bucket, key)
	if err != nil {
		log.Println("Veritabanindan silinemedi:", err)
		sendJSONresponse(w, 500, "Error", "Database connection error")
		return
	}

	// DELETE sorgusu gonderince aranan şarta (WHERE bucket = $1 AND key = $2) uyan satirlari arar.
	// eger o isimde bir kova bulursa satiri siler. ben 1 satiri sildim diye cevap doner.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Veritabani hesaplama hatasi.", err)
		sendJSONresponse(w, 500, "Error", "Server error, operation could not be verified")
		return
	}

	if rowsAffected == 0 {
		sendJSONresponse(w, 404, "Error", "File not found")
		return
	}

	err = os.Remove(fullPath)
	if err != nil {
		log.Printf("Veritabani silindi fakat disk silinemedi: %s - Hata: %v\n", fullPath, err)
	}

	sendJSONresponse(w, 200, "Success", "File deleted successfully")
}

func listObjects(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucket := r.PathValue("bucket")
	rows, err := db.Query(`SELECT key, size, content_type, created_at FROM objects WHERE bucket = $1`, bucket)
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Failed to list objects")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/json")

	dosyalar := []ObjectInfo{}

	for rows.Next() {
		var dosya ObjectInfo
		err := rows.Scan(&dosya.Key, &dosya.Size, &dosya.ContentType, &dosya.CreatedAt)
		if err != nil {
			log.Println("Dosya okunamadi", err)
			continue
		}
		dosyalar = append(dosyalar, dosya)
	}
	if err := rows.Err(); err != nil {
		log.Println("Döngü sırasında hata oluştu:", err)
	}
	err = json.NewEncoder(w).Encode(dosyalar)
	if err != nil {
		log.Println("JSON encode hatasi.", err)
	}

}

func generatePresignedURL(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	//10 dk deadline verdik
	expiresInSec := int64(600)

	// gecerlilik suresini ekleyip stringe ceviriyoruz
	expiresAt := time.Now().Unix() + expiresInSec
	expiresStr := fmt.Sprintf("%d", expiresAt)

	apiKey := os.Getenv("API_KEY")

	// 4 argumani da hashlenmek uzere fonksiyona veriyoruz
	signature := createSignature(bucket, key, expiresStr, apiKey)

	// kullaniciya url yi veriyoruz.
	presignedURL := fmt.Sprintf("http://localhost:8080/buckets/%s/objects/%s?expires=%s&signature=%s",
		bucket, key, expiresStr, signature)

	sendJSONresponse(w, 200, "Success", presignedURL)
}
