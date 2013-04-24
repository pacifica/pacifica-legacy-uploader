set PATH=%PATH%;c:\wix
del pacificauploader.msi
del pacificauploaderui.msi
del pacificauploadersdk.msi
heat dir "go\src\pacificauploaderd\ui" -cg PacificaUploaderUI -gg -sfrag -dr INSTALLDIR -var var.DaemonUI -out pacificauploaderuigen.wxs || goto :error
candle -ext WiXNetFxExtension pacificauploader.wxs pacificauploaderuigen.wxs -dDaemonUI=go\src\pacificauploaderd\ui -ext WixUtilExtension || goto :error
light -ext WiXNetFxExtension -out pacificauploader.msi pacificauploader.wixobj pacificauploaderuigen.wixobj -ext WixUtilExtension || goto :error
candle pacificauploaderui.wxs || goto :error
light -out pacificauploaderui.msi pacificauploaderui.wixobj -ext WixUtilExtension || goto :error
candle pacificauploadersdk.wxs || goto :error
light -out pacificauploadersdk.msi pacificauploadersdk.wixobj || goto :error
exit /b 0

:error
echo Failed with error #%errorlevel%.
exit /b %errorlevel%
