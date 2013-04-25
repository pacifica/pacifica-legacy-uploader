package web

import (
	"encoding/xml"
	"log"
	"pacificauploaderd/common"
	"net/http"
	"strings"	
)

var defaultUI *uiServer
var ServMux *http.ServeMux

type UpdateInfo struct {
	Update     bool
	UpdatePath string
}

type uiServer struct {
}

type fileHandler struct {
	handler http.Handler
	prefix  string
}

func (fh *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if strings.HasPrefix(p, fh.prefix) {
		p = p[len(fh.prefix):]
		r.URL.Path = p
	}
	fh.handler.ServeHTTP(w, r)
}

func FileServer(handler http.Handler, prefix string) http.Handler {
	return &fileHandler{handler, prefix}
}

func update(w http.ResponseWriter, req *http.Request) {
	u := &UpdateInfo{Update: common.UpdateDownloaded, UpdatePath: common.UpdatePath}
	m, err := xml.Marshal(u)
	if err != nil {
		log.Printf("Error marshalling %+v, %+v", u, err)
		//HTTP 404
		http.NotFound(w, req)
		return
	}
	w.Write(m)
}

func webServerInit() {
	defaultUI = new(uiServer)

	ServMux = http.NewServeMux();

	ServMux.Handle("/status/", http.RedirectHandler("/ui/status.html", http.StatusMovedPermanently))
	ServMux.HandleFunc("/auth/", authHandle)
	ServMux.Handle("/ui/", FileServer(http.FileServer(http.Dir(common.UiDirGet())), "/ui"))
	ServMux.HandleFunc("/update/", update)
}

func ServerRun() {
	http.ListenAndServe("localhost:39999", ServMux)
}
