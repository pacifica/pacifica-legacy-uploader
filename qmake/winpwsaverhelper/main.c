#ifndef __reserved
#define __reserved
#endif

#include <windows.h>
#include <msi.h>
#include <msiquery.h>
#include <wincrypt.h>
#include <stdio.h>
#include <stdarg.h>

#define BUFFSIZE 1024

void pacifica_msi_log(MSIHANDLE hInstall, const WCHAR *str, ...)
{
	va_list args;
	WCHAR buffer[10240];
	UINT err;
	MSIHANDLE record = MsiCreateRecord(1);
	va_start(args, str);
	vswprintf(buffer, str, args);
	va_end(args);
	if(record == ERROR_INVALID_HANDLE)
	{
		return;
	}
	err = MsiRecordSetString(record, 0, buffer);
	if(err == ERROR_SUCCESS)
	{
		MsiProcessMessage(hInstall, INSTALLMESSAGE_ERROR, record);
	}
	MsiCloseHandle(record);
}

UINT __stdcall SavePW(MSIHANDLE hInstall)
{
	HANDLE handle;
	DWORD size;
	WCHAR wbuffer[BUFFSIZE + 1] = {};
	WCHAR *pw;
	WCHAR path[MAX_PATH] = {};
	DATA_BLOB blob;
	DATA_BLOB blob_out;
	DATA_BLOB entropy;
	int res;
	size = BUFFSIZE;
	/* Custom Action Data should be "[CommonAppDataFolder];[GENEDPW]" */
	res = MsiGetPropertyW(hInstall, L"CustomActionData", wbuffer, &size);
	if(res != ERROR_SUCCESS)
	{
		pacifica_msi_log(hInstall, TEXT("Failed to get CustomActionData property. %lu %lu."), res, GetLastError());
		return ERROR_INSTALL_FAILURE;
	}
	//DEBUGGING
	//pacifica_msi_log(hInstall, TEXT("Got CustomActionData."));

	/* Find a pointer to the first ';' in CustomActionData */
	pw = wcschr(wbuffer, L';');

	/* Null terminate after CommonAppDataFolder */
	*pw = L'\0';

	/* Advance to the beginning of the password */
	pw++;

	/* Put CommonAppDataFolder in path */
	wcscpy(path, wbuffer);

	/* Full path to password file */
	wcscat(path, L"Pacifica\\Uploader\\priv\\localservice.cred");

	//DEBUGGING
	//pacifica_msi_log(hInstall, TEXT("Got Path %s"), path);

	handle = CreateFileW(path, GENERIC_WRITE, 0, NULL, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);

	if(handle == INVALID_HANDLE_VALUE)
	{
		pacifica_msi_log(hInstall, TEXT("Failed to open password cred file. %s %lu."), path, GetLastError());
		return ERROR_INSTALL_FAILURE;
	}

	/* Put password in blob */
	blob.cbData = wcslen(pw) * sizeof(WCHAR);
	blob.pbData = (void*)pw;

	entropy.pbData = (void*)"a*5%K!M0jn,(vJ19Kz/.nf9031";
	entropy.cbData = wcslen(((WCHAR*)entropy.pbData));

	/* Encrypt password blob */
	res = CryptProtectData(&blob, NULL, &entropy, NULL, NULL, CRYPTPROTECT_UI_FORBIDDEN, &blob_out);
	if(!res)
	{
		pacifica_msi_log(hInstall, TEXT("Failed to crypt data. %lu."), GetLastError());
		return ERROR_INSTALL_FAILURE;
	}

	/* Write encrypted password blob to file */
	WriteFile(handle, blob_out.pbData, blob_out.cbData, &size, NULL);

	/* Cleanup */
	CloseHandle(handle);
	LocalFree(blob_out.pbData);

	return ERROR_SUCCESS;
}
