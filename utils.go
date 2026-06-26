package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var validBucketName = regexp.MustCompile(`^[a-z0-9-]+$`)  //ai
var validObjectKey = regexp.MustCompile(`^[[:print:]]+$`) // ai

type ObjectInfo struct {
	Key         string    `json:"key"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}
type BucketInfo struct {
	Buckets []string `json:"buckets"`
}

type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func sendJSONresponse(w http.ResponseWriter, statusCode int, durum string, mesaj string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(APIResponse{
		Status:  durum,
		Message: mesaj,
	})
	if err != nil {
		log.Println("JSON response gonderilemedi:", err)
	}
}

// guvenlik amacli path kontrolu, kod tekrari olmamasi icin fonksiyon olarak yazildi.
func pathControl(kovaAdi, dosyaYolu string) (string, error) {
	if strings.TrimSpace(kovaAdi) == "" {
		return "", fmt.Errorf("bucket adi bos olamaz veya bosluklardan olusamaz") //bosluk kontrolu basta yapilsin
	}
	if len(kovaAdi) < 3 || len(kovaAdi) > 63 {
		return "", fmt.Errorf("kova adi en az 3, en fazla 63 karakter olmalidir") //uzunluk kontrolu
	}
	if !validBucketName.MatchString(kovaAdi) {
		return "", fmt.Errorf("kova adi sadece kucuk harf, rakam ve tire icerebilir") //matchstring dogruysa true degilse false doner
	}

	//hasprefix on eke bakar, verilen stringi basta arar
	//hasuffix son eke bakar, verilen stringi sonda arar
	//contains icinde arar , verilen stringi icinde arar
	//s3 klonu olmak acisindan regex kurallarini da aldim.
	if strings.HasPrefix(kovaAdi, "-") || strings.HasSuffix(kovaAdi, "-") || strings.Contains(kovaAdi, "--") {
		return "", fmt.Errorf("kova adi tire ile baslayamaz, bitemez veya iki tire yanyana olamaz")
	}
	fullPath := filepath.Clean(filepath.Join(baseStorageDir, kovaAdi, dosyaYolu))
	bucketPath := filepath.Clean(filepath.Join(baseStorageDir, kovaAdi))
	if fullPath != bucketPath && !strings.HasPrefix(fullPath, bucketPath+string(os.PathSeparator)) {
		return "", fmt.Errorf("geçersiz yol")
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
