#ifndef __reserved
#define __reserved
#endif

#include <windows.h>
#include <msi.h>
#include <msiquery.h>
#include <wincrypt.h>

#define BUFFSIZE 24
UINT __stdcall GenPW(MSIHANDLE hInstall)
{
	int i;
	HCRYPTPROV provider = 0;
	BYTE b;
	BYTE buffer[BUFFSIZE] = {};
	WCHAR wbuffer[BUFFSIZE + 1] = {};
//FIXME error check these.
	CryptAcquireContextW(&provider, 0, 0, PROV_RSA_FULL, CRYPT_VERIFYCONTEXT);
	CryptGenRandom(provider, BUFFSIZE, buffer);
	CryptReleaseContext(provider, 0);
	for(i = 0; i < BUFFSIZE; i++)
	{
		wbuffer[i] = ' ' + (buffer[i] % ('~' - ' ' + 1));
	}
	wbuffer[i] = '\0';
	MsiSetPropertyW(hInstall, TEXT("GENEDPW"), wbuffer);
	return ERROR_SUCCESS;
}
