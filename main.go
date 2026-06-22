package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

var baseStorageDir string

// next http.handlerfunc hedeftir, sifre dogruysa nereye gilicegini soyluyoruz. kisaca w,r parametresi alan standart bir fonksiyon
// http.handlerfunc zaten arkada func(w,r) seklinde, kullanim kolayligi acisindan bu sekilde yazilmis.
// mux bizden calistirilacak bir fonksiyon istiyor, biz ise araya bir middleware koyarak ciktisina mux'un alacagi fonksiyonu ayarliyoruz
// mux gelen isteği alır -> mux bizim firewall'un urettigi fonksiyonu çağırır
// -> fonksiyon sifreye bakar
// ->sifre dogruysa asil hedef olan 'next(w,r)' cagirilir ve islem gerceklesir."
// mux.HandleFunc("GET /buckets", firewall(apiKey, func(w, r) { ... }))
func firewall(API_KEY string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientKey := r.Header.Get("X-API-Key")
		if clientKey != API_KEY {
			http.Error(w, "Gecersiz API anahtari", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env dosyası yüklenirken hata")
	}
	baseStorageDir = os.Getenv("STORAGE_DIR")
	if baseStorageDir == "" {
		log.Fatal("STORAGE_DIR degiskeni tanimlanmalidir")
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY degiskeni tanimlanmalidir")
	}

	db := initDB()          //db initialiton
	defer db.Close()        //main kapaninca dosyayi kapatir
	createTable(db)         //tablo olusturma
	go garbageCollector(db) //temizlikci goroutine

	//deafult olursa guvenlik sorunu olabilirdi, mux handle'i yazildi
	mux := http.NewServeMux()

	//Fonksiyonlara dependency injection eklendi, cunku mux 2 parametre bekliyordu ve db ekleyemiyordum. bundan sonra db parametresine de uygun sekilde calisir.
	//Firewall isimli
	//object(dosya) listeler
	//GET /buckets/{bucket}/objects/{key...} rotasından once yaz. yoksa mux, readObject'e yönlendirir.
	mux.HandleFunc("GET /buckets/{bucket}/objects", firewall(apiKey,
		func(w http.ResponseWriter, r *http.Request) {
			listObjects(w, r, db)
		}))

	mux.HandleFunc("PUT /buckets/{bucket}/objects/{key...}", firewall(apiKey,
		func(w http.ResponseWriter, r *http.Request) {
			addObject(w, r, db)
		}))

	mux.HandleFunc("DELETE /buckets/{bucket}/objects/{key...}", firewall(apiKey,
		func(w http.ResponseWriter, r *http.Request) {
			deleteObject(w, r, db)
		}))

	mux.HandleFunc("GET /buckets/{bucket}/objects/{key...}", firewall(apiKey,
		func(w http.ResponseWriter, r *http.Request) {
			readObject(w, r, db)
		}))

	mux.HandleFunc("GET /buckets", firewall(apiKey,
		func(w http.ResponseWriter, r *http.Request) {
			listBuckets(w, r, db)
		}))

	fmt.Println("Sistem 8080 portunda çalışıyor...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
