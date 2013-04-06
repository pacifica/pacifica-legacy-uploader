QMAKE_CFLAGS_RELEASE = -static
QMAKE_LFLAGS_RELEASE = -static

QT =
TARGET = pacificauploaderservice
CONFIG += release

win32 {
	SOURCES = service.c
	target.path = ../../build/
	LIBS     += -ladvapi32 -lShlwapi
	INSTALLS += target
}
