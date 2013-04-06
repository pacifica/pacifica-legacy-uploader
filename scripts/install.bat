start /wait msiexec /i pacificauploadersdk.msi /qn /l* "C:\Temp\pacificauploadersdk.install.log"
start msiexec /i pacificauploader.msi /qn /l*v "C:\Temp\pacificauploader.install.log"
