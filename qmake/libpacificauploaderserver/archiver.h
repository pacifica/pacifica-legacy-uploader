#ifdef __cplusplus
extern "C" {
#endif

#ifdef WIN32
#ifdef BUILDING_PACIFICA_UPLOADER_SERVER_DLL
#define TOEXPORT __declspec(dllexport)
#else
#define TOEXPORT __declspec(dllimport)
#endif
#else
#define TOEXPORT
#endif

TOEXPORT int pacifica_uploader_server_archiver(int tarFD, int fileFD, const char * srcName, long size, long mtime);

#ifdef __cplusplus
}
#endif
