start /wait msiexec /x pacificauploadersdk.msi /qn /l* "C:\Temp\pacificauploadersdk.uninstall.log"
start msiexec /x pacificauploader.msi /qn /l* "C:\Temp\pacificauploader.uninstall.log"
