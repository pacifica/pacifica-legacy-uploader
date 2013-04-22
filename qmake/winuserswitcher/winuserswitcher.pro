QMAKE_CFLAGS_RELEASE = -static -m32
QMAKE_LFLAGS_RELEASE = -static -m32

QT =
TARGET = pacificauploaderuserswitcher
CONFIG += release
CONFIG += release console

win32 {
	SOURCES = main.c
	target.path = ../../build/
	LIBS     += -ladvapi32 -lcrypt32
	INSTALLS += target
}
