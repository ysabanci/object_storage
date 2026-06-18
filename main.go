package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const baseStorageDir = "/Users/ysabanci/Desktop/s3" //sabit dizin

func objeEkle(w http.ResponseWriter, r *http.Request) { //dosya ekleme handleri

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

	_, err = io.Copy(newFile, r.Body)
	if err != nil {
		http.Error(w, "Dosya yükleme hatası", http.StatusInternalServerError)
		return
	} //exception handling

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Yaratma işlemi gerçekleştirildi.")
}

func objeOku(w http.ResponseWriter, r *http.Request) {
	fullPath, err := pathControl(r.PathValue("bucket"), r.PathValue("key"))
	if err != nil {
		http.Error(w, "Geçersiz yol", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, fullPath)

}

func objeSil(w http.ResponseWriter, r *http.Request) { //dosya silen fonksiyon
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
	mux := http.NewServeMux() //deafult olursa guvenlik sorunu olabilirdi, mux handle'i yazildi
	mux.HandleFunc("PUT /buckets/{bucket}/objects/{key...}", objeEkle)
	mux.HandleFunc("GET /buckets/{bucket}/objects/{key...}", objeOku)
	mux.HandleFunc("DELETE /buckets/{bucket}/objects/{key...}", objeSil)
	fmt.Println("Sistem 8080 portunda çalışıyor...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
