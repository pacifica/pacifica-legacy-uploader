To test this service run:

sc.exe create "Pacifica Uploader" binPath= FULL_QUOTED_PATH_TO_FILE start= auto

Then 

sc.exe start "Pacifica Uploader"

To uninstall:

sc.exe stop "Pacifica Uploader"

Then

sc.exe delete "Pacifica Uploader"