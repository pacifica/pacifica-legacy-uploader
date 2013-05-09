TEMPLATE = app
#QT = core
TARGET = pacificauploaderservice
SOURCES  = main.cpp
#QTSERVICE_LIBNAME = QtSolutions_Service
#CONFIG += debug console qt link_pkgconfig
#CONFIG += release link_pkgconfig

CONFIG += release link_pkgconfig
PKGCONFIG   = qtsolutionsservice
INSTALLS += target

win32 {
	target.path = "..\\..\\build"
}
