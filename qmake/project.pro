TEMPLATE = subdirs
SUBDIRS = gui
unix {
	SUBDIRS += unixhelper unixuserswitcher uuidgen
}
win32 {
	SUBDIRS += winpwhelper winpwsaverhelper winuserswitcher uuidgen
}
#win32 {
#	SUBDIRS += winhelper
#}
