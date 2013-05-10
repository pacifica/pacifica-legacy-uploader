TEMPLATE = subdirs
SUBDIRS = gui libpacificauploaderserver
unix {
	SUBDIRS += unixhelper unixuserswitcher uuidgen
}
win32 {
	#SUBDIRS += winpwhelper winpwsaverhelper winuserswitcher uuidgen uploaderservice
	SUBDIRS += uuidgen uploaderservice
}
#win32 {
#	SUBDIRS += winhelper
#}
