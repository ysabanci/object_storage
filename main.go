package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const baseStorageDir = "/Users/ysabanci/Desktop/s3"

func objeEkle(w http.ResponseWriter, r *http.Request) {
	kovaAdi := r.PathValue("bucket")
	dosyaYolu := r.PathValue("key")

	fmt.Println("Gelen Kova:", kovaAdi)
	fmt.Println("Gelen Dosya Yolu:", dosyaYolu)

	fullPath := filepath.Join(baseStorageDir, kovaAdi, dosyaYolu)
	folderPath := filepath.Dir(fullPath)

	err := os.MkdirAll(folderPath, 0775)
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
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Yaratma işlemi gerçekleştirildi.")
}

func objeOku(w http.ResponseWriter, r *http.Request) {
	kovaAdi := r.PathValue("bucket")
	dosyaYolu := r.PathValue("key")

	fullPath := filepath.Join(baseStorageDir, kovaAdi, dosyaYolu)
	http.ServeFile(w, r, fullPath)
}

func objeSil(w http.ResponseWriter, r *http.Request) {
	kovaAdi := r.PathValue("bucket")
	dosyaYolu := r.PathValue("key")

	fullPath := filepath.Join(baseStorageDir, kovaAdi, dosyaYolu)

	err := os.Remove(fullPath)
	if err != nil {
		http.Error(w, "Dosya silinemedi veya bulunamadı", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /buckets/{bucket}/objects/{key...}", objeEkle)
	mux.HandleFunc("GET /buckets/{bucket}/objects/{key...}", objeOku)
	mux.HandleFunc("DELETE /buckets/{bucket}/objects/{key...}", objeSil)
	fmt.Println("Sistem 8080 portunda çalışıyor...")
	log.Fatal(http.ListenAndServe(":8080", mux))
} //github
