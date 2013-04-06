set PATH=%PATH%;c:\wix
del %PACIFICASDK%build\pacificauploadersetup.exe
candle -ext WiXNetFxExtension -out %PACIFICASDK%build\pacificauploaderall.wixobj %PACIFICASDK%build\pacificauploaderall.wxs
light -ext WiXNetFxExtension -out %PACIFICASDK%build\pacificauploadersetup.exe %PACIFICASDK%build\pacificauploaderall.wixobj -ext WixBalExtension -dWixStdbaLicenseUrl=http://my.emsl.pnl.gov/myemsl/license/uploader
