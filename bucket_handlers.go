package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// addObject fonksiyonumuz zaten MkdirAll sayesinde dosyaya ait tum pathi olusturur. bos bir kova olusturmak istenirse
// diye bu fonksiyonu yazdim.
func addBucket(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucket := r.PathValue("bucket")
	folderPath, err := pathControl(bucket, "")
	if err != nil {
		sendJSONresponse(w, 400, "Error", "Invalid bucket path")
		return
	}
	err = os.MkdirAll(folderPath, 0775)
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Failed to create bucket")
		return
	}

	query := `
	INSERT INTO buckets (name) VALUES ($1) ON CONFLICT (name) DO NOTHING
`
	_, err = db.Exec(query, bucket)
	if err != nil {
		log.Println("Bucket veritabanina eklenemedi:", err)
		sendJSONresponse(w, 500, "Error", "Bucket created on disk but could not be saved to database")
		return
	}
	sendJSONresponse(w, 201, "Success", "Bucket created successfully")
}

func deleteBucket(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucket := r.PathValue("bucket")
	fullPath, err := pathControl(bucket, "")

	if err != nil {
		sendJSONresponse(w, 400, "Error", "Invalid bucket path") //revize deleteonject ile ayni hata
		return
	}

	var objectCount int

	err = db.QueryRow(`SELECT COUNT(*) FROM objects WHERE bucket = $1`, bucket).Scan(&objectCount)
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Database connection error")
		return
	}

	if objectCount > 0 {
		sendJSONresponse(w, 409, "Error", "BucketNotEmpty: The bucket you tried to delete is not empty")
		return
	}
	query := `DELETE FROM buckets WHERE name = $1`
	result, err := db.Exec(query, bucket)
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Database connection error")
		return
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Server error, operation could not be verified")
		return
	}
	if rowsAffected == 0 {
		sendJSONresponse(w, 404, "Error", "Bucket not found")
		return
	}
	err = os.Remove(fullPath)
	if err != nil {
		log.Printf("Bucket veritabanindan silindi fakat diskten silinemedi: %s - Hata: %v\n", fullPath, err)
	}
	sendJSONresponse(w, 200, "Success", "Bucket deleted successfully")
}

func listBuckets(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.Query(`SELECT name FROM buckets ORDER BY name ASC`)
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Failed to list buckets")
		return
	}
	defer rows.Close()

	buckets := []string{}
	for rows.Next() {
		var bucket string
		err := rows.Scan(&bucket)
		if err != nil {
			log.Println("Bucket okunamadi", err)
			continue
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		log.Println("Döngü sırasında hata oluştu:", err)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(BucketInfo{Buckets: buckets})
	if err != nil {
		log.Println("JSON encode hatasi.", err)
	}
}

func updateBucketVisibility(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucket := r.PathValue("bucket")

	publicStr := r.URL.Query().Get("public")
	if publicStr != "true" && publicStr != "false" {
		sendJSONresponse(w, 400, "Error", "Invalid 'public' parameter. It must be 'true' or 'false'")
		return
	}
	isPublic := publicStr == "true"

	query := "UPDATE buckets SET is_public = $1 WHERE name = $2"
	result, err := db.Exec(query, isPublic, bucket)

	if err != nil {
		log.Println("Bucket visibility guncellenemedi:", err)
		sendJSONresponse(w, 500, "Error", "Database connection error")
		return
	}

	//kova var mi yok mu kontrolu
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		sendJSONresponse(w, 500, "Error", "Server error, operation could not be verified")
		return
	}

	if rowsAffected == 0 {
		sendJSONresponse(w, 404, "Error", "Bucket not found")
		return
	}

	var statusMsg string
	if isPublic {
		statusMsg = "Bucket is now PUBLIC. Anyone can read its objects."
	} else {
		statusMsg = "Bucket is now PRIVATE. API Key is required."
	}

	sendJSONresponse(w, 200, "Success", statusMsg)

}
