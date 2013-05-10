QT += network xml webkit
TARGET = pacificauploaderstatus
FORMS += pacificauploaderabout.ui simplebrowser.ui
HEADERS = status.h simplebrowser.h reusablewindow.h
SOURCES = main.cpp status.cpp simplebrowser.cpp reusablewindow.cpp
RESOURCES += resource.qrc
CONFIG += link_pkgconfig
PKGCONFIG += pacificaauth

unix {
	target.path = /usr/libexec/pacifica
	autostart.files = pacificauploaderstatus.desktop
	autostart.path = /etc/xdg/autostart
	INSTALLS += autostart
}
win32 {
	CONFIG += release
	target.path = "..\\..\\build\\release"
}

INSTALLS += target
