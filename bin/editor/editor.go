package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
)

//go:generate go-bindata -pkg main -prefix assets/ assets/...

var httpFlag = flag.String("http", ":8080", "Listen for HTTP connections on this address.")
var mainTemplate *template.Template

func init() {
	flag.Parse()

	mt, err := Asset("assets/templates/main.template.html")
	if err != nil {
		log.Fatal(err)
	}
	mainTemplate, err = template.New("main").Parse(string(mt))
	if err != nil {
		log.Fatal(err)
	}
}

func assetHandler(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	for len(path) > 0 && path[0:1] == "/" {
		path = path[1:]
	}
	ass, err := Asset(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	llp := len(path)
	mime := "text/plain"
	if llp > 4 && path[llp-4:] == ".css" {
		mime = "text/css"
	} else if llp > 3 && path[llp-3:] == ":js" {
		mime = "application/javascript"
	} else {
		mime = http.DetectContentType(ass)
	}

	w.Header().Set("Content-Type", mime)

	w.Write(ass)
}

func mainHandler(w http.ResponseWriter, req *http.Request) {
	var data = struct {}{}

	err := mainTemplate.Execute(w, data)
	if err != nil {
		log.Println("t.Execute:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
func main() {
	http.HandleFunc("/assets/", assetHandler)
	http.HandleFunc("/", mainHandler)

	err := http.ListenAndServe(*httpFlag, nil)
	if err != nil {
		log.Fatalln("ListenAndServe:", err)
	}
}
