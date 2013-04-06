TARGET = uuidgen 
CONFIG += release console

SOURCES = main.cpp
INSTALLS += target
win32 {
	target.path = ../../build/
}
unix {
	target.path = /usr/libexec/pacifica/
}
