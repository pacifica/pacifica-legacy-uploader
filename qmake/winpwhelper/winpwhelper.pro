QMAKE_CFLAGS_RELEASE = -static -I../../msisdk
QMAKE_LFLAGS_RELEASE = -static -Wl,--kill-at

QT =
TARGET = winpwhelper
TEMPLATE = lib
CONFIG += release dll

win32 {
	SOURCES = main.c
	target.path = ../../build/
	LIBS     += -ladvapi32 -lShlwapi ../../msisdk/msi.lib
	INSTALLS += target
}
