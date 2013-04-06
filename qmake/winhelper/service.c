#ifdef UNICODE
#undef UNICODE
#endif
#include <windows.h>
#include <stdio.h>
#include <tchar.h>
#include <time.h>
#include <string.h>
#include <stdlib.h>
#include <malloc.h>
#include "Shlwapi.h"

#define PROCESS_FILE_NAME _T("pacificauploaderd.exe")
//define ARGUMENTS _T("-system 1> \"C:\\PacificaUploaderLog.txt\" 2>&1")
#define ARGUMENTS _T("-system")
#define UPDATE_TIME 1000  /* One second between updates */

/* Windows Service Callbacks */
VOID WINAPI ServerCtrlHandler(DWORD);
VOID WINAPI ServiceMain(DWORD argc, LPTSTR argv[]);

/* Prototypes */
int ServiceSpecific(int, LPTSTR *);
void UpdateStatus(int, int);
void LogEvent(LPCTSTR, WORD);
BOOL LogInit(LPTSTR);
void LogClose();

static BOOL shutDown = FALSE;
static SERVICE_STATUS hServStatus;
static SERVICE_STATUS_HANDLE hSStat; /* handle for setting status */
static LPTSTR serviceName = _T("Pacifica Uploader");
static LPTSTR logFileName = _T(".\\LogFiles\\PacificaServiceLog.txt");  //TODO, figure out where to put this.

/* Main entry point */
int _tmain(int argc, LPTSTR argv[])
{
	if(!LogInit(logFileName)) {
		return -1;
	}

	SERVICE_TABLE_ENTRY DispatchTable[] =
	{
		{ serviceName, ServiceMain},
		{ NULL, NULL }
	};

	StartServiceCtrlDispatcher(DispatchTable);

	LogClose();

	return 0;
}

/* ServiceMain entry point, called by main program. */
VOID WINAPI ServiceMain(DWORD argc, LPTSTR argv[])
{
	//TODO - remove NGT 2/13/2012
	LogEvent(_T("Entering ServiceMain."), EVENTLOG_SUCCESS);

	hServStatus.dwServiceType = SERVICE_WIN32_OWN_PROCESS;
	hServStatus.dwCurrentState = SERVICE_START_PENDING;
	hServStatus.dwControlsAccepted = SERVICE_ACCEPT_STOP |
	  SERVICE_ACCEPT_SHUTDOWN;
	hServStatus.dwWin32ExitCode = NO_ERROR;
	hServStatus.dwServiceSpecificExitCode = 0;
	hServStatus.dwCheckPoint = 0;
	hServStatus.dwWaitHint = 2 * UPDATE_TIME;

	hSStat = RegisterServiceCtrlHandler(serviceName, ServerCtrlHandler);

	if (hSStat == 0) {
		LogEvent(_T("Cannot register handler"), EVENTLOG_ERROR_TYPE);
		hServStatus.dwCurrentState = SERVICE_STOPPED;
		hServStatus.dwWin32ExitCode = ERROR_SERVICE_SPECIFIC_ERROR;
		hServStatus.dwServiceSpecificExitCode = 1;
		UpdateStatus(SERVICE_STOPPED, -1);
		return;
	}

	//TODO - remove NGT 2/13/2012
	//LogEvent(_T("Control handler registered"), EVENTLOG_SUCCESS);
	SetServiceStatus(hSStat, &hServStatus);
	//TODO - remove NGT 2/13/2012
	//LogEvent(_T("Status SERVICE_START_PENDING"), EVENTLOG_SUCCESS);

	ServiceSpecific(argc, argv);

	//TODO - remove NGT 2/13/2012
	//LogEvent(_T("Spin shut down"), EVENTLOG_SUCCESS);
	/*  The service has been completed, and will shutdown. */
	UpdateStatus(SERVICE_STOPPED, 0);
	LogEvent(_T("Status set to SERVICE_STOPPED"), EVENTLOG_SUCCESS);
	return;
}

/* Logic specific to this service.  Starts uploader and then spins. */
int ServiceSpecific(int argc, LPTSTR argv[])
{
	LogEvent(_T("Entering ServiceSpecific."), EVENTLOG_SUCCESS);

	UpdateStatus(-1, -1);

 	//TODO - should probably adjust size, make global or malloc/free as needed. For now, it's just big :)
	TCHAR errorString[1024];
	DWORD maxFileNameLength;
	LPSTR servicePath;
	LPSTR uploaderPath;
	size_t newSize;
	STARTUPINFO si;
	PROCESS_INFORMATION pi;

	memset(errorString, 0, sizeof(TCHAR)*1024);
	maxFileNameLength = MAX_PATH * sizeof(TCHAR);
	servicePath = malloc(maxFileNameLength);

	//Get the absolute path to the executing service.
	if(!GetModuleFileName(NULL, servicePath, maxFileNameLength))
	{
		LogEvent(_T("Failed to get location of this service path. Cannot find process to execute."), EVENTLOG_ERROR_TYPE);
		LogEvent(_T("Server process will shut down."), EVENTLOG_ERROR_TYPE);
		return 0;
	}

	//Remove file name from path.
	PathRemoveFileSpec(servicePath);

	//Get size of full path to uploader process.
	newSize = _tcslen(_T("\""))*2 + //*2 for a quote on each end.
			  _tcslen(servicePath) +
			  _tcslen(_T("\\")) +
			  _tcslen(PROCESS_FILE_NAME) +
			  1; //+1 for null termination

	uploaderPath = malloc(newSize + strlen(ARGUMENTS) + 1);
	memset(uploaderPath, 0, newSize);
	//uploaderPath is quoted since it could have spaces
	_stprintf(uploaderPath, _T("\"%s\\%s\" %s"), servicePath, PROCESS_FILE_NAME, ARGUMENTS);

	if(newSize > maxFileNameLength)
	{
		LogEvent(_T("Process name too long."), EVENTLOG_ERROR_TYPE);
	}

	/*newSize = _tcslen(_T("\""))*2 +
			  _tcslen(fileName) +
			  _tcslen(_T("\\")) +
			  _tcslen(PROCESS_FILE_NAME) +
			  _tcslen(_T(" ")) +
			  _tcslen(ARGUMENTS);*/

	//LPSTR cmdLine = malloc(_tcslen(ARGUMENTS));
	//_stprintf(cmdLine, _T("\"%s\\%s\" %s"), fileName, PROCESS_FILE_NAME, ARGUMENTS);

	memset(&si, 0, sizeof(STARTUPINFO));
	memset(&pi, 0, sizeof(PROCESS_INFORMATION));

	//LogEvent(cmdLine, EVENTLOG_INFORMATION_TYPE);
	LogEvent(uploaderPath, EVENTLOG_INFORMATION_TYPE);
	LogEvent(ARGUMENTS, EVENTLOG_INFORMATION_TYPE);

	if(!CreateProcess(NULL, uploaderPath, NULL, NULL,
		FALSE, CREATE_NO_WINDOW, NULL, NULL, &si, &pi))
	{
		_stprintf(errorString, _T("Failed to start process %s %s error %u"), uploaderPath, ARGUMENTS, GetLastError());
		LogEvent(_T(errorString), EVENTLOG_ERROR_TYPE);
		LogEvent(_T("Failed to start process."), EVENTLOG_ERROR_TYPE);
		LogEvent(_T("Server process will shut down."), EVENTLOG_ERROR_TYPE);
		return 0;
	}

	_stprintf(errorString, _T("PID %u"), pi.dwProcessId);
	LogEvent(errorString, EVENTLOG_INFORMATION_TYPE);

	//TODO remove NGT 2/14/2012
	LogEvent(_T("Made it past CreateProcess."), EVENTLOG_INFORMATION_TYPE);

	//TODO one of these statements crashes the program.  Figure out...for now, we leak two buffers...
	//free(fileName);
	//free(cmdLine);

	//TODO remove NGT 2/14/2012
	LogEvent(_T("Made it past free's."), EVENTLOG_INFORMATION_TYPE);

	/* Service is now running */
	UpdateStatus(SERVICE_RUNNING, -1);
	//TODO - remove NGT 2/13/2012
	//LogEvent(_T("Status update. Service running"), EVENTLOG_SUCCESS);

	LogEvent(_T("Starting main service loop"), EVENTLOG_SUCCESS);
	/* Update the status periodically. */
	while (!shutDown) {
		Sleep (UPDATE_TIME);
	  	UpdateStatus(-1, -1); /* Assume no change */
		//TODO - remove NGT 2/13/2012
		//LogEvent(_T("Status update. No change"), EVENTLOG_SUCCESS);
	}
	//TODO - remove NGT 2/13/2012
	//LogEvent(_T("Server process has shut down."), EVENTLOG_SUCCESS);

	//TODO - signal pacificauploaderd.exe to shutdown and/or kill it.
	//http://www.codeguru.com/forum/showthread.php?t=312449
	//http://blogs.msdn.com/b/oldnewthing/archive/2004/07/22/191123.aspx
	//http://drdobbs.com/184416547

	return 0;
}

/* Control Handler Function */
VOID WINAPI ServerCtrlHandler(DWORD dwControl)
{
	switch (dwControl) {
		case SERVICE_CONTROL_SHUTDOWN:
		case SERVICE_CONTROL_STOP:
			shutDown = TRUE;/* Set the global shutDown flag */
			UpdateStatus(SERVICE_STOP_PENDING, -1);
			break;
		case SERVICE_CONTROL_PAUSE:
			//Not implemented
			break;
		case SERVICE_CONTROL_CONTINUE:
			//Not implemented
			break;
		case SERVICE_CONTROL_INTERROGATE:
			//Not implemented
			break;
		default:
			if(dwControl > 127 && dwControl < 256) {
				//User defined control here
			}
			break;
	}
	UpdateStatus (-1, -1);
	return;
}

/*
 *  Sets service status and checkpoint (specific value or increment)
 *  If Check is less than 0, increment checkpoint, otherwise checkpoint
 *  is set to Check.
 */
void UpdateStatus(int NewStatus, int Check)
{
	if (Check < 0 ) {
		hServStatus.dwCheckPoint++;
	} else {
		hServStatus.dwCheckPoint = Check;
	}

	if (NewStatus >= 0) {
		hServStatus.dwCurrentState = NewStatus;
	}

	if (!SetServiceStatus(hSStat, &hServStatus)) {
		LogEvent(_T("Cannot set status"), EVENTLOG_ERROR_TYPE);
		hServStatus.dwCurrentState = SERVICE_STOPPED;
		hServStatus.dwWin32ExitCode = ERROR_SERVICE_SPECIFIC_ERROR;
		hServStatus.dwServiceSpecificExitCode = 2;
		UpdateStatus(SERVICE_STOPPED, -1);
		return;
	}/* else {
		//TODO - remove NGT 2/14/2012
		//LogEvent(_T("Service Status updated."), EVENTLOG_SUCCESS);
	}*/

	return;
}

static FILE * logFp = NULL;
void LogEvent(LPCTSTR UserMessage, WORD type)
{
	TCHAR cTimeString[30] = _T("");
	time_t currentTime = time(NULL);
	_tcsncat(cTimeString, _tctime(&currentTime), 30);
	/* Remove the new line at the end of the time string */
	cTimeString[_tcslen(cTimeString)-2] = _T('\0');
	_ftprintf(logFp, _T("%s. "), cTimeString);

	if (type == EVENTLOG_SUCCESS || type == EVENTLOG_INFORMATION_TYPE) {
		_ftprintf(logFp, _T("%s"), _T("Information. "));
	} else if (type == EVENTLOG_ERROR_TYPE) {
		_ftprintf(logFp, _T("%s"), _T("Error.       "));
	} else if (type == EVENTLOG_WARNING_TYPE) {
		_ftprintf(logFp, _T("%s"), _T("Warning.     "));
	} else {
		_ftprintf(logFp, _T("%s"), _T("Unknown.     "));
	}

	_ftprintf(logFp, _T("%s\n"), UserMessage);
	fflush(logFp);

	return;
}

BOOL LogInit(LPTSTR name)
{
   logFp = _tfopen (name, _T("a+"));
   if (logFp != NULL) {
	   LogEvent(_T("Initialized Logging"), EVENTLOG_SUCCESS);
   }
   return (logFp != NULL);
}

void LogClose()
{
   LogEvent(_T("Closing Log"), EVENTLOG_SUCCESS);
   return;
}