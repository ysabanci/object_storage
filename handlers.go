package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var validBucketName = regexp.MustCompile(`^[a-z0-9-]+$`)  //ai
var validObjectKey = regexp.MustCompile(`^[[:print:]]+$`) // ai
type ObjectInfo struct {
	Key         string    `json:"dosya_adi"`
	Size        int64     `json:"dosya_boyutu"`
	ContentType string    `json:"dosya_tipi"`
	CreatedAt   time.Time `json:"dosya_tarihi"`
}
type BucketInfo struct {
	Buckets []string `json:"bucketlar"`
}

type APIResponse struct {
	Durum string `json:"durum"`
	Mesaj string `json:"mesaj"`
}

func sendJSONresponse(w http.ResponseWriter, statusCode int, durum string, mesaj string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(APIResponse{
		Durum: durum,
		Mesaj: mesaj,
	})
	if err != nil {
		log.Println("JSON response gonderilemedi:", err)
	}
}

func addObject(w http.ResponseWriter, r *http.Request, db *sql.DB) { //dosya ekleme handleri

	bucket := r.PathValue("bucket")
	key := r.PathValue("key")

	fullPath, err := pathControl(bucket, key)
	if err != nil {
		sendJSONresponse(w, 400, "Hata", "Gecersiz yol")
		return
	}
	folderPath := filepath.Dir(fullPath)

	var exist bool

	err = db.QueryRow(`
	SELECT EXISTS(SELECT 1 FROM buckets WHERE name = $1)`, bucket).Scan(&exist)

	if err != nil {
		log.Println("Bucket sorgusu hatasi:", err)
		sendJSONresponse(w, 500, "Hata", "Veritabani hatasi.")
		return
	}

	if !exist {
		sendJSONresponse(w, 404, "Hata", "Bucket mevcut degil, once kova olusturulmali.")
		return
	}

	err = os.MkdirAll(folderPath, 0775) //folderPathi tum yolu baz alarak komple olusturur.
	if err != nil {
		sendJSONresponse(w, 500, "Hata", "Klasor olusturma hatasi")
		return
	}

	newFile, err := os.Create(fullPath)
	if err != nil {
		sendJSONresponse(w, 500, "Hata", "Dosya olusturma hatasi")
		return
	}
	defer newFile.Close()

	size, err := io.Copy(newFile, r.Body)
	if err != nil {
		sendJSONresponse(w, 500, "Hata", "Dosya yukleme hatasi")
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
		sendJSONresponse(w, 500, "Hata", "Dosya diske yazıldı fakat metadata veritabanina eklenemedi.")
	}

	sendJSONresponse(w, 201, "Basarili", "Dosya basariyla eklendi.")
}

// addObject fonksiyonumuz zaten MkdirAll sayesinde dosyaya ait tum pathi olusturur. bos bir kova olusturmak istenirse
// diye bu fonksiyonu yazdim.
func addBucket(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucket := r.PathValue("bucket")
	folderPath, err := pathControl(bucket, "")
	if err != nil {
		sendJSONresponse(w, 400, "Hata", "Gecersiz bucket yolu.")
		return
	}
	err = os.MkdirAll(folderPath, 0775)
	if err != nil {
		sendJSONresponse(w, 500, "Hata", "Kova olusturulamadi.")
		return
	}

	query := `
	INSERT INTO buckets (name) VALUES ($1) ON CONFLICT (name) DO NOTHING
`
	_, err = db.Exec(query, bucket)
	if err != nil {
		log.Println("Bucket veritabanina eklenemedi:", err)
		sendJSONresponse(w, 500, "Hata", "Kova diskte olusturuldu ancak veritabanina kaydedilemedi.")
		return
	}
	sendJSONresponse(w, 201, "Basarili", "Bucket basariyla olusturuldu.")
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
		isAuthorized := (clientKey == apiKey)

		if !isAuthorized { //presigned url kontrolu
			expires := r.URL.Query().Get("expires")
			signature := r.URL.Query().Get("signature")

			if expires != "" && signature != "" {
				expiresInt, err := strconv.ParseInt(expires, 10, 64)
				if err != nil {
					sendJSONresponse(w, 400, "Hata", "Gecersiz zaman formati. Lutfen gecerli bir format kullanin.")
					return
				}
				if time.Now().Unix() > expiresInt {
					sendJSONresponse(w, 401, "Hata", "Linkin suresi dolmus.")
					return
				}
				expectedSignature := createSignature(bucket, key, expires, apiKey)
				if signature == expectedSignature {
					isAuthorized = true
				}
			}
		}
		if !isAuthorized { //erisim reddi buradadir
			sendJSONresponse(w, 401, "Hata", "Erisim reddedildi: Gecersiz yetki")
			return
		}
	}

	fullPath, err := pathControl(r.PathValue("bucket"), r.PathValue("key"))
	if err != nil {
		sendJSONresponse(w, 400, "Hata", "Gecersiz yol")
		return
	}
	var exist bool //sonra varlik dogrulanir
	
	query := `SELECT EXISTS (SELECT 1 FROM objects WHERE bucket = $1 AND key = $2)`
	err = db.QueryRow(query, bucket, key).Scan(&exist)
	if err != nil {
		log.Println("Veritabanindan okunamadi:", err)
		sendJSONresponse(w, 500, "Hata", "Veritabani baglanti hatasi.")
		return
	}
	if !exist {
		sendJSONresponse(w, 404, "Hata", "Object not exist")
		return
	}

	http.ServeFile(w, r, fullPath)

}

func deleteObject(w http.ResponseWriter, r *http.Request, db *sql.DB) { //dosya silen fonksiyon
	bucket := r.PathValue("bucket")
	key := r.PathValue("key")
	fullPath, err := pathControl(bucket, key)
	if err != nil {
		sendJSONresponse(w, 400, "Hata", "Gecersiz bucket yolu.")
		return
	}

	query := `DELETE FROM objects WHERE bucket = $1 AND key = $2`
	result, err := db.Exec(query, bucket, key)
	if err != nil {
		log.Println("Veritabanindan silinemedi:", err)
		sendJSONresponse(w, 500, "Hata", "Veritabani baglanti hatasi.")
		return
	}

	// DELETE sorgusu gonderince aranan şarta (WHERE bucket = $1 AND key = $2) uyan satirlari arar.
	// eger o isimde bir kova bulursa satiri siler. ben 1 satiri sildim diye cevap doner.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println("Veritabani hesaplama hatasi.", err)
		sendJSONresponse(w, 500, "Hata", "Sunucu hatasi, islem dogrulanamadi.")
		return
	}

	if rowsAffected == 0 {
		sendJSONresponse(w, 404, "Hata", "Boyle bir dosya bulunamadi.")
		return
	}

	err = os.Remove(fullPath)
	if err != nil {
		log.Printf("Veritabani silindi fakat disk silinemedi: %s - Hata: %v\n", fullPath, err)
	}

	sendJSONresponse(w, 200, "Basarili", "Dosya basariyla silindi.")
}

func listBuckets(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.Query(`SELECT name FROM buckets ORDER BY name ASC`)
	if err != nil {
		sendJSONresponse(w, 500, "Hata", "Bucketlar listelenemedi.")
		return
	}
	defer rows.Close()

	var buckets []string
	for rows.Next() {
		var bucket string
		err := rows.Scan(&bucket)
		if err != nil {
			log.Println("Bucket okunamadi", err)
			continue
		}
		buckets = append(buckets, bucket)
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(BucketInfo{Buckets: buckets})
	if err != nil {
		log.Println("JSON encode hatasi.", err)
	}
}

func listObjects(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	bucket := r.PathValue("bucket")
	rows, err := db.Query(`SELECT key, size, content_type, created_at FROM objects WHERE bucket = $1`, bucket)
	if err != nil {
		sendJSONresponse(w, 500, "Hata", "Objectler listelenemedi.")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/json")

	var dosyalar []ObjectInfo

	for rows.Next() {
		var dosya ObjectInfo
		err := rows.Scan(&dosya.Key, &dosya.Size, &dosya.ContentType, &dosya.CreatedAt)
		if err != nil {
			log.Println("Dosya okunamadi", err)
			continue
		}
		dosyalar = append(dosyalar, dosya)
	}
	err = json.NewEncoder(w).Encode(dosyalar)
	if err != nil {
		log.Println("JSON encode hatasi.", err)
	}

}

// guvenlik amacli path kontrolu, kod tekrari olmamasi icin fonksiyon olarak yazildi.
func pathControl(kovaAdi, dosyaYolu string) (string, error) {
	if strings.TrimSpace(kovaAdi) == "" {
		return "", fmt.Errorf("Bucket adi bos olamaz veya bosluklardan olusamaz.") //bosluk kontrolu basta yapilsin
	}
	if len(kovaAdi) < 3 || len(kovaAdi) > 63 {
		return "", fmt.Errorf("Kova adi en az 3, en fazla 63 karakter olmalidir.") //uzunluk kontrolu
	}
	if !validBucketName.MatchString(kovaAdi) {
		return "", fmt.Errorf("Kova adi sadece kucuk harf, rakam ve tire icerebilir.") //matchstring dogruysa true degilse false doner
	}

	//hasprefix on eke bakar, verilen stringi basta arar
	//hasuffix son eke bakar, verilen stringi sonda arar
	//contains icinde arar , verilen stringi icinde arar
	//s3 klonu olmak acisindan regex kurallarini da aldim.
	if strings.HasPrefix(kovaAdi, "-") || strings.HasSuffix(kovaAdi, "-") || strings.Contains(kovaAdi, "--") {
		return "", fmt.Errorf("Kova adi tire ile baslayamaz, bitemez veya iki tire yanyana olamaz.")
	}
	fullPath := filepath.Join(baseStorageDir, kovaAdi, dosyaYolu)
	if !strings.HasPrefix(fullPath, filepath.Clean(baseStorageDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("geçersiz yol:")
	}
	return fullPath, nil
}

// veriler birbirine karismasin (collision) diye aralarina belirgin bir ayrac koymak zorundayiz.
func createSignature(bucket, key, expires, apiKey string) string {
	message := fmt.Sprintf("%s:%s:%s", bucket, key, expires)

	h := hmac.New(sha256.New, []byte(apiKey))
	h.Write([]byte(message))

	return hex.EncodeToString(h.Sum(nil))

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

	// kullaniciya urlyi veriyoruz.
	presignedURL := fmt.Sprintf("http://localhost:8080/buckets/%s/objects/%s?expires=%s&signature=%s",
		bucket, key, expiresStr, signature)

	sendJSONresponse(w, 200, "Basarili", presignedURL)
}
