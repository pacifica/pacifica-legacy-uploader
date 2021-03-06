QMAKE_CFLAGS_RELEASE = -static -I../../msisdk
QMAKE_LFLAGS_RELEASE = -static -Wl,--kill-at

QT =
TARGET = winpwsaverhelper
TEMPLATE = lib
CONFIG += release dll

win32 {
	SOURCES = main.c
	target.path = "..\\..\\build\\release"
	LIBS     += -ladvapi32 -lshlwapi -lcrypt32 ../../msisdk/msi.lib
	INSTALLS += target
}
