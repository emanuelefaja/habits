package web

import (
	"net/http"
)

// SetupStaticFileHandlers configures all static file serving routes
func SetupStaticFileHandlers() {
	// Static file directories
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.Handle("/icons/", http.StripPrefix("/icons/", http.FileServer(http.Dir("static/icons"))))
	http.Handle("/content/media/", http.StripPrefix("/content/media/", http.FileServer(http.Dir("content/media"))))
	http.Handle("/brand/", http.StripPrefix("/brand/", http.FileServer(http.Dir("static/brand"))))

	// Manifest and Service Worker
	http.HandleFunc("/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		serveStaticFileWithContentType(w, r, "static/manifest.json", "application/manifest+json")
	})
	http.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Service-Worker-Allowed", "/")
		http.ServeFile(w, r, "static/sw.js")
	})

	// Sitemap
	http.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		serveStaticFileWithContentType(w, r, "static/sitemap.xml", "application/xml")
	})
}

// serveStaticFileWithContentType serves a static file with the specified content type
func serveStaticFileWithContentType(w http.ResponseWriter, r *http.Request, filePath, contentType string) {
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, filePath)
}
