#include <windows.h>
#include <winbase.h>
#include <wincrypt.h>
#include <stdio.h>
#include <malloc.h>

//FIXME stat file, build buffer.
#define BUFFSIZE 10240


//FIXME resource leak plug.
char *pw_get(const char *filename)
{
	int i;
	char *retval;
	WCHAR *str;
	BOOL ok;
	DWORD len;
	BYTE buffer[BUFFSIZE];
	DATA_BLOB blob;
	DATA_BLOB blob_out;
	DATA_BLOB entropy;
	HANDLE handle;
	entropy.pbData = (void*)"a*5%K!M0jn,(vJ19Kz/.nf9031";
	entropy.cbData = wcslen(((WCHAR*)entropy.pbData));
	handle = CreateFileA(filename, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, 0, NULL);
	if(handle == INVALID_HANDLE_VALUE)
	{
		fprintf(stderr, "Failed to open password file. %lu\n", GetLastError());
		return NULL;
	}
	ok = ReadFile(handle, buffer, BUFFSIZE, &len, NULL);
	if(!ok)
	{
		fprintf(stderr, "Failed to read password file. %lu\n", GetLastError());
		return NULL;
	}
/*	fprintf(stderr, "Read %lu bytes.\n", len);*/
	blob.cbData = len;
	blob.pbData = (void*)buffer;
	ok = CryptUnprotectData(&blob, NULL, &entropy, NULL, NULL, CRYPTPROTECT_UI_FORBIDDEN, &blob_out);
	if(!ok)
	{
		fprintf(stderr, "Failed to unprotect data. %lu\n", GetLastError());
		return NULL;
	}
/*	fprintf(stderr, "Decrypted %lu bytes.\n", blob_out.cbData);*/
	retval = malloc(blob_out.cbData / sizeof(WCHAR) + 1);
	if(!retval)
	{
		fprintf(stderr, "Failed to malloc.\n");
		return NULL;
	}
	str = (WCHAR*)(blob_out.pbData);
	for(i = 0; i < blob_out.cbData / sizeof(WCHAR); i++)
	{
		retval[i] = (char)(str[i]);
	}
	retval[blob_out.cbData / sizeof(WCHAR)] = '\0';
	return retval;
}

int pacifica_switch_process_user(char *user, char *pw, char *program)
{
	DWORD len;
	HANDLE token;
	PROCESS_INFORMATION pi;
	STARTUPINFOA si;
	memset(&si, 0, sizeof(STARTUPINFO));
	si.cb = sizeof(STARTUPINFO);
	si.lpDesktop = "";
//FIXME Still need to pull this out of storage somehow...
	int res = LogonUserA(user, ".", pw, LOGON32_LOGON_SERVICE, LOGON32_PROVIDER_DEFAULT, &token);
	if(res == 0)
	{
		return GetLastError();
	}
	res = CreateProcessAsUserA(token, NULL, program, NULL, NULL, TRUE, 0, NULL, NULL, &si, &pi);
	if(res == 0)
	{
		return GetLastError();
	}
	res = WaitForSingleObject(pi.hProcess, INFINITE);
	if(res == 0)
	{
		return GetLastError();
	}
	res = GetExitCodeProcess(pi.hProcess, &len);
	if(res == 0)
	{
		return GetLastError();
	}
	return 0;
}

void usage(const char *program)
{
	fprintf(stderr, "Usage:\n");
	fprintf(stderr, "\t%s -u user -p file program.\n", program);
	exit(-1);
}

int main(int argc, char *argv[])
{
	int res;
	char *pw;
	if(argc <= 5 || strcmp(argv[1], "-u") || strcmp(argv[3], "-p"))
	{
		usage(argv[0]);
	}
	pw = pw_get(argv[4]);
	if(!pw)
	{
		fprintf(stderr, "Failed to get password.\n");
		return -1;
	}
	res = pacifica_switch_process_user(argv[2], pw, argv[5]);
	return res;
}
