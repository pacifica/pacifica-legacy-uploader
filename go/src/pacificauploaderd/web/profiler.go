package web

import "log"
import "pacificauploaderd/common"
import "net/http/pprof"
import "net/http"

func profilerInit() {
	if common.Profiler {
		log.Printf("Profiler loaded.\n")
		ServMux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		ServMux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		ServMux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		ServMux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	}
}
