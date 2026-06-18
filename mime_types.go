package main

import "mime"

func init() { //bazi bilinemeyecek olan content/type'lari manuel olarak ekledim.
	// === FOTOĞRAFLAR ===
	mime.AddExtensionType(".heic", "image/heic")
	mime.AddExtensionType(".heif", "image/heif")
	mime.AddExtensionType(".avif", "image/avif")
	mime.AddExtensionType(".webp", "image/webp")
	//mime.AddExtensionType(".svg", "image/svg+xml") xml icerir, js gomulebilir ve tehlikeli olabilir
	mime.AddExtensionType(".bmp", "image/bmp")
	mime.AddExtensionType(".ico", "image/x-icon")
	mime.AddExtensionType(".tiff", "image/tiff")
	mime.AddExtensionType(".tif", "image/tiff")
	mime.AddExtensionType(".raw", "image/x-raw")

	// === VİDEOLAR ===
	mime.AddExtensionType(".mkv", "video/x-matroska")
	mime.AddExtensionType(".mov", "video/quicktime")
	mime.AddExtensionType(".avi", "video/x-msvideo")
	mime.AddExtensionType(".wmv", "video/x-ms-wmv")
	mime.AddExtensionType(".flv", "video/x-flv")
	mime.AddExtensionType(".m4v", "video/x-m4v")
	mime.AddExtensionType(".ts", "video/mp2t")
	mime.AddExtensionType(".3gp", "video/3gpp")

	// === SES DOSYALARI ===
	mime.AddExtensionType(".flac", "audio/flac")
	mime.AddExtensionType(".aac", "audio/aac")
	mime.AddExtensionType(".wma", "audio/x-ms-wma")
	mime.AddExtensionType(".m4a", "audio/mp4")
	mime.AddExtensionType(".opus", "audio/opus")
	mime.AddExtensionType(".aiff", "audio/aiff")

	// === BELGELER ===
	mime.AddExtensionType(".docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	mime.AddExtensionType(".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	mime.AddExtensionType(".pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	mime.AddExtensionType(".odt", "application/vnd.oasis.opendocument.text")
	mime.AddExtensionType(".ods", "application/vnd.oasis.opendocument.spreadsheet")
	mime.AddExtensionType(".epub", "application/epub+zip")
	mime.AddExtensionType(".pages", "application/x-iwork-pages-sffpages")
	mime.AddExtensionType(".numbers", "application/x-iwork-numbers-sffnumbers")
	mime.AddExtensionType(".key", "application/x-iwork-keynote-sffkey")

	// === ARŞİVLER (Sadece arşiv, çalıştırılamaz) ===
	mime.AddExtensionType(".7z", "application/x-7z-compressed")
	mime.AddExtensionType(".rar", "application/vnd.rar")
	mime.AddExtensionType(".tar", "application/x-tar")
	mime.AddExtensionType(".gz", "application/gzip")
	mime.AddExtensionType(".bz2", "application/x-bzip2")

	// === FONTLAR ===
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".woff", "font/woff")
	mime.AddExtensionType(".otf", "font/otf")
	mime.AddExtensionType(".ttf", "font/ttf")
}
