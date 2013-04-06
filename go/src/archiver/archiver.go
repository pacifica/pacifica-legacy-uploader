// Package archiver provides tar capabilies.
package archiver
/*
#cgo LDFLAGS: -larchive
#cgo windows CFLAGS: -fno-stack-check -fno-stack-protector -mno-stack-arg-probe

#define _XOPEN_SOURCE 500
#include <unistd.h>
#include <errno.h>
#include <malloc.h>
#include <fcntl.h>
#include <string.h>

#include <archive.h>
#include <archive_entry.h>

#ifdef WIN32
#define read _read
#define write _write
#endif

typedef struct {
	int archiveFD;
	int error;
} CallbackData;

static int header_open(struct archive *a, void *data)
{
	return ARCHIVE_OK;
}

static __LA_SSIZE_T header_write(struct archive *a, void *data, const void *buff, size_t size)
{
	CallbackData *cbd = (CallbackData*)data;
	int res;
	// header info requested in write_header
	res = write(cbd->archiveFD, buff, size);
	if(res == -1)
	{
		cbd->error = -1;
	}
	return res;
}

static int header_close(struct archive *a, void *data)
{
	return 0;
}

#define BUFFSIZE 1024 * 1024
int archive(int tarFD, int fileFD, const char * srcName, long size, long mtime )
{
	int                      res;
	CallbackData             cbd;
	struct archive         * archiveWriter  = NULL;
	archive_open_callback  * aoc            = header_open;
	archive_write_callback * awc            = header_write;
	archive_close_callback * acc            = header_close;
	struct archive_entry   * entry;
	int                      len;
	char                     buffer[BUFFSIZE];
	cbd.error = 0;
	cbd.archiveFD = tarFD;

	archiveWriter       = archive_write_new();
	if(!archiveWriter)
	{
		return -ENOMEM;
	}
	//NEEDED so we can strip off the extra padding that gets added.
	archive_write_set_bytes_per_block(archiveWriter, 512);
	archive_write_set_compression_none(archiveWriter);
	archive_write_set_format_pax(archiveWriter);
        archive_write_open(archiveWriter, &cbd, aoc, awc, acc);

        entry                = archive_entry_new();
	archive_entry_set_pathname(entry, srcName);
	archive_entry_set_filetype(entry, AE_IFREG);
	archive_entry_set_perm(entry, 0644);
	archive_entry_set_size(entry, size);
	archive_entry_set_mtime(entry, mtime, 0);
	archive_entry_set_ctime(entry, mtime, 0);
	archive_entry_set_atime(entry, mtime, 0);
	res = archive_write_header(archiveWriter, entry);
	if(res == -1)
	{
		goto cleanup;
	}

	len = read(fileFD, buffer, BUFFSIZE);
	if(len == -1)
	{
		res = -1;
		goto cleanup;
	}
	while(len > 0)
	{
		res = archive_write_data(archiveWriter, buffer, len);
		if(res == -1)
		{
			goto cleanup;
		}
		len = read(fileFD, buffer, BUFFSIZE);
		if(len == -1)
		{
			res = -1;
			goto cleanup;
		}
	}
cleanup:
	archive_entry_free(entry);
	archive_write_close(archiveWriter);
	return 0; 
}

void write_EOT(int fd, long size)
{
   char eot_block[1024];
   long extra = 512 - size % 512;
   if(extra > 511)
   {
      extra = 0;
   }

   memset(eot_block, 0, 1024);
   if(extra)
   {
      write(fd, eot_block, extra);
   }
   write(fd, eot_block, 1024);
}
*/
import "C"
import (
	"unsafe"
)

func write_EOT(fd int, size int64) {
    C.write_EOT(C.int(fd), C.long(size))
}

func archive(tar int, sourcefile int, src string, size int64, mtime int64) int {
    srcName := C.CString(src)
    defer C.free(unsafe.Pointer(srcName))

    error := C.archive(C.int(tar), C.int(sourcefile), srcName, C.long(size), C.long(mtime))
    return int(error)
}

