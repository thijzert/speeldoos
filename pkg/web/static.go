package web

import "net/http"

func (*Server) serveStaticAsset(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) < 9 || r.URL.Path[0:8] != "/assets/" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	assetPath := r.URL.Path[8:]
	l := len(assetPath)
	if len(assetPath) < 6 || assetPath[0:5] != "dist/" {
		// This may be okay in a development build
		if assetsEmbedded {
			http.Error(w, "Please just look up the source code on Github", http.StatusForbidden)
			return
		}
	}

	b, err := getAsset(assetPath)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if l > 5 {
		contentType := ""
		if assetPath[l-5:] == ".html" {
			contentType = "text/html"
		} else if assetPath[l-3:] == ".js" {
			contentType = "application/javascript"
		} else if assetPath[l-4:] == ".css" {
			contentType = "text/css"
		} else if assetPath[l-4:] == ".svg" {
			contentType = "image/svg+xml"
		} else if assetPath[l-4:] == ".ico" {
			contentType = "image/vnd.microsoft.icon"
		}

		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
	}
	w.Write(b)
}
