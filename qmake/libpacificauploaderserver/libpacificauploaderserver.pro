QMAKE_CFLAGS_DEBUG = -DBUILDING_PACIFICA_UPLOADER_SERVER_DLL -g -O0
QT =
TARGET = pacificauploaderserver
TEMPLATE = lib
CONFIG  += dll debug
DEFINES =
QMAKE_LINK = gcc

DESTDIR = debug
OBJ_DIR = debug

HEADERS = archiver.h
SOURCES = archiver.c

LIBS    += -larchive

unix:contains(QMAKE_HOST.arch, x86_64): {
	target.path = /usr/lib64
}
unix:!contains(QMAKE_HOST.arch, x86_64): {
	target.path = /usr/lib
}
win32 {
	target.path = "..\\..\\build"
}

INSTALLS += target
