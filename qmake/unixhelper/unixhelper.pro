QT +=
TARGET = daemonize

unix {
	SOURCES = main.c
	target.path = /usr/libexec/pacifica/
	initd.files = pacificauploaderd
	initd.path = /etc/init.d/
	INSTALLS += target initd
}
