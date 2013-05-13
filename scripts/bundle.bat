set PATH=%PATH%;c:\wix

rem Copy pacificauploaderservice.exe out of the release dir so that heat doesn't include
mv "build\release\pacificauploaderservice.exe" "build"

heat dir "go\src\pacificauploaderd\ui" -cg PacificaUploaderUI -gg -sfrag -dr INSTALLDIR -var var.DaemonUI -out pacificauploaderuigen.wxs || goto :error
heat dir "build\release" -srd -cg PacificaUploaderGroup -gg -sfrag -dr INSTALLDIR -var var.PacificaUploaderGroup -out pacificauploadergen.wxs

rem Move pacificauploaderservice.exe back in the release directory.
mv "build\pacificauploaderservice.exe" "build\release"

candle pacificauploader.wxs pacificauploadergen.wxs pacificauploaderuigen.wxs -dDaemonUI=go\src\pacificauploaderd\ui -dPacificaUploaderGroup=build\release -ext WixUtilExtension || goto :error
light -out pacificauploader.msi pacificauploader.wixobj pacificauploadergen.wixobj pacificauploaderuigen.wixobj -ext WixUtilExtension || goto :error

candle pacificauploaderui.wxs || goto :error
light -out pacificauploaderui.msi pacificauploaderui.wixobj -ext WixUtilExtension || goto :error
candle pacificauploadersdk.wxs || goto :error
light -out pacificauploadersdk.msi pacificauploadersdk.wixobj || goto :error

exit /b 0

:error
echo Failed with error #%errorlevel%.
exit /b %errorlevel%
