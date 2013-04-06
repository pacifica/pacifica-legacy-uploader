QT +=
TARGET = userswitcher

unix {
	SOURCES = main.c
	target.path = /usr/libexec/pacifica/
	INSTALLS += target
}
