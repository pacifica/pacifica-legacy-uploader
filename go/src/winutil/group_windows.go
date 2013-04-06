package winutil

/*
#cgo LDFLAGS: -lNetapi32

#include <windows.h>
#include <stdio.h>
#include <string.h>
#include <Lm.h>
#include <Lmapibuf.h>
#include <Lmaccess.h>

//Returns 1 if user (userName) is a member if of group (groupName),
//otherwise returns 0.
int IsInLocalGroupC(const char* userName, const char* groupName) {
	//For reference, the function this function wraps
	//NET_API_STATUS NetLocalGroupGetMembers(
	//  __in     LPCWSTR servername,
	//  __in     LPCWSTR localgroupname,
	//  __in     DWORD level,
	//  __out    LPBYTE *bufptr,
	//  __in     DWORD prefmaxlen,
	//  __out    LPDWORD entriesread,
	//  __out    LPDWORD totalentries,
	//  __inout  PDWORD_PTR resumehandle
	//);
	
	size_t userSize = strlen(userName) * sizeof(wchar_t);
	wchar_t* userNameW = malloc(userSize);
	
	size_t groupSize = strlen(groupName) * sizeof(wchar_t);
	wchar_t* groupNameW = malloc(groupSize);
	
	mbstowcs(userNameW, userName, userSize);
	
	mbstowcs(groupNameW, groupName, groupSize);

	//Debugging
	//wprintf(L"userName == %s\r\n", userNameW);
	//wprintf(L"groupName == %s\r\n", groupNameW);
	
	DWORD level;
	level = 3;
		
	LPLOCALGROUP_MEMBERS_INFO_3 bufptr;
	bufptr = NULL;
	
	DWORD prefixmaxlen;
	prefixmaxlen = MAX_PREFERRED_LENGTH;
	
	DWORD entriesread;
	entriesread = 0;
	
	DWORD totalentries;
	totalentries = 0;
	
	NET_API_STATUS status;
	status = NetLocalGroupGetMembers(NULL, groupNameW, level, (LPBYTE*)&bufptr,
		prefixmaxlen, &entriesread, &totalentries, NULL);
	
	//Debugging
	//wprintf(L"status is %ld\r\n", status);
	
	int ret;
	if (status == NERR_Success) {
		ret = 0;

		//Loop though bufptr and look for userNameW in it.
		DWORD i;
		for (i = 0; i < entriesread; i++) {
			LOCALGROUP_MEMBERS_INFO_3 entry = bufptr[i];

			//Debugging
			//wprintf(L"%s is member of %s\r\n", entry.lgrmi3_domainandname, groupNameW);

			if (_wcsicmp(userNameW, entry.lgrmi3_domainandname) == 0) {
				ret = 1;
				break;
			}
		}
	} else {
		//TODO - it would be nice to return error information instead of just TRUE/FALSE
		//a char* with a return code description that can be converted to an error type in Go...
		ret = 0;	
	}
		
	free(userNameW);
	free(groupNameW);
	NetApiBufferFree(bufptr);
	
	return ret;
}
*/
import "C"

import (
	"unsafe"
)

//Return true if user (userName) is a member of group (groupName), returns
//false otherwise.
func IsInLocalGroup(userName, groupName string) (bool, error) {
	cUserName := C.CString(userName)
	defer C.free(unsafe.Pointer(cUserName))
	
	cGroupName := C.CString(groupName)
	defer C.free(unsafe.Pointer(cGroupName))
	
	resultC := C.IsInLocalGroupC(cUserName, cGroupName)
	result := int(resultC)
	if result != 0 {
		return true, nil
	}
	return false, nil
}
