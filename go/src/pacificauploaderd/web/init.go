package web

import "log"

func Init() {
	log.Println("Web subsystem init.")
	authInit()
	webServerInit()
}
