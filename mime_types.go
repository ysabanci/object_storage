package main

import "mime"

// bazi bilinemeyecek olan content/type'lari manuel olarak ekledim.
func init() {
	// === FOTOĞRAFLAR ===
	_ = mime.AddExtensionType(".heic", "image/heic")
	_ = mime.AddExtensionType(".heif", "image/heif")
	_ = mime.AddExtensionType(".avif", "image/avif")
	_ = mime.AddExtensionType(".webp", "image/webp")
	// mime.AddExtensionType(".svg", "image/svg+xml") xml icerir, js gomulebilir ve tehlikeli olabilir
	_ = mime.AddExtensionType(".bmp", "image/bmp")
	_ = mime.AddExtensionType(".ico", "image/x-icon")
	_ = mime.AddExtensionType(".tiff", "image/tiff")
	_ = mime.AddExtensionType(".tif", "image/tiff")
	_ = mime.AddExtensionType(".raw", "image/x-raw")

	// === VİDEOLAR ===
	_ = mime.AddExtensionType(".mkv", "video/x-matroska")
	_ = mime.AddExtensionType(".mov", "video/quicktime")
	_ = mime.AddExtensionType(".avi", "video/x-msvideo")
	_ = mime.AddExtensionType(".wmv", "video/x-ms-wmv")
	_ = mime.AddExtensionType(".flv", "video/x-flv")
	_ = mime.AddExtensionType(".m4v", "video/x-m4v")
	_ = mime.AddExtensionType(".ts", "video/mp2t")
	_ = mime.AddExtensionType(".3gp", "video/3gpp")

	// === SES DOSYALARI ===
	_ = mime.AddExtensionType(".flac", "audio/flac")
	_ = mime.AddExtensionType(".aac", "audio/aac")
	_ = mime.AddExtensionType(".wma", "audio/x-ms-wma")
	_ = mime.AddExtensionType(".m4a", "audio/mp4")
	_ = mime.AddExtensionType(".opus", "audio/opus")
	_ = mime.AddExtensionType(".aiff", "audio/aiff")

	// === BELGELER ===
	_ = mime.AddExtensionType(".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	_ = mime.AddExtensionType(".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	_ = mime.AddExtensionType(".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	_ = mime.AddExtensionType(".odt", "application/vnd.oasis.opendocument.text")
	_ = mime.AddExtensionType(".ods", "application/vnd.oasis.opendocument.spreadsheet")
	_ = mime.AddExtensionType(".epub", "application/epub+zip")
	_ = mime.AddExtensionType(".pages", "application/x-iwork-pages-sffpages")
	_ = mime.AddExtensionType(".numbers", "application/x-iwork-numbers-sffnumbers")
	_ = mime.AddExtensionType(".key", "application/x-iwork-keynote-sffkey")

	// === ARŞİVLER (Sadece arşiv, çalıştırılamaz) ===
	_ = mime.AddExtensionType(".7z", "application/x-7z-compressed")
	_ = mime.AddExtensionType(".rar", "application/vnd.rar")
	_ = mime.AddExtensionType(".tar", "application/x-tar")
	_ = mime.AddExtensionType(".gz", "application/gzip")
	_ = mime.AddExtensionType(".bz2", "application/x-bzip2")

	// === FONTLAR ===
	_ = mime.AddExtensionType(".woff2", "font/woff2")
	_ = mime.AddExtensionType(".woff", "font/woff")
	_ = mime.AddExtensionType(".otf", "font/otf")
	_ = mime.AddExtensionType(".ttf", "font/ttf")
}
