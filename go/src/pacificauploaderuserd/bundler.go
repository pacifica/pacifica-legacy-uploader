package main

import (
	"io"
	"os"
	"fmt"
	"errors"
	"crypto/sha1"
	"archiver"
	userdrpc "pacificauploaderuserd/rpc"
)

const (
	BUFFSIZE int = 1024
	ENDSIZE int64 = 512*2
)

var zeroEnd = make([]byte, ENDSIZE)

func IsPermission(err error) bool {
	if os.IsPermission(err) {
		return true
//FIXME Stupid broken Go!
	} else if err.(*os.PathError).Err.Error() == "Access is denied." {
		return true
	}
	return false
}

func bundleReset(w *os.File, size int64) error {
	pending_offset := size-ENDSIZE
	if size == 0 {
		pending_offset = 0
	}
	offset, err := w.Seek(pending_offset, 0)
	if err != nil {
		return err
	}
	if offset != pending_offset {
		return errors.New("Failed to seek!")
	}
	err = w.Truncate(size)
	if err != nil {
		return err
	}
	if size == 0 {
		return nil
	}
	n, err := w.Write(zeroEnd)
	if err != nil || n != len(zeroEnd) {
		return errors.New("Failed to write end block")
	}
	return nil
}

func bundleResetRPC(w *os.File, bundleInitialSize int64, reply *userdrpc.BundleFileResult, etype userdrpc.BundleFileError) {
	err := bundleReset(w, bundleInitialSize)
	(*reply).Error = etype
	if err != nil {
		(*reply).Error = userdrpc.BundleFileError_CORRUPT
	}
}

func panicIfErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Userd: %v\n", err)
		panic("Error")
	}
}

func (s Server) BundleFile(args *userdrpc.BundleFileArgs, reply *userdrpc.BundleFileResult) error {
	var buffer [BUFFSIZE]byte
	*reply = userdrpc.BundleFileResult{Retval: false, Sha1: "", Error: userdrpc.BundleFileError_UNKNOWN}
	w, err := os.OpenFile(args.BundlePath, os.O_WRONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			(*reply).Error = userdrpc.BundleFileError_BNF
		} else if IsPermission(err) {
			(*reply).Error = userdrpc.BundleFileError_BPERM
		}
		return nil
	}
	defer w.Close()
	info, err := os.Lstat(args.BundlePath)
	if err != nil {
		if os.IsNotExist(err) {
			(*reply).Error = userdrpc.BundleFileError_BNF
		} else if IsPermission(err) {
			(*reply).Error = userdrpc.BundleFileError_BPERM
		}
		return nil
	}
	bundleInitialSize := info.Size()
	if bundleInitialSize > 0 {
		offset, err := w.Seek(bundleInitialSize-ENDSIZE, 0)
		fmt.Fprintf(os.Stderr, "Seeked to %v\n", offset)
		if err != nil {
			(*reply).Error = userdrpc.BundleFileError_UNKNOWN
			return nil
		}
	}
// w is opened on line 74
//	tw := tar.NewWriter(w)
	hash := sha1.New()
	fmt.Fprintf(os.Stderr, "Got file request %v %v %v\n", args.BundlePath, args.LocalPath, args.Name)
// r is the file descriptor for file to be archived
	r, err := os.Open(args.LocalPath)
	if err != nil {
		if os.IsNotExist(err) {
			(*reply).Error = userdrpc.BundleFileError_FNF
		} else if IsPermission(err) {
			(*reply).Error = userdrpc.BundleFileError_PERM
		} else {
			fmt.Fprintf(os.Stderr, "Got unknown error in open local path. %v\n", err)
		}
		return nil
	}
	defer r.Close()
//FIXME if symlink, error.
// info is file info for file to be archived
	info, err = os.Lstat(args.LocalPath)
	if err != nil {
		if os.IsNotExist(err) {
			(*reply).Error = userdrpc.BundleFileError_FNF
		} else if IsPermission(err) {
			(*reply).Error = userdrpc.BundleFileError_PERM
		} else {
			fmt.Fprintf(os.Stderr, "Got unknown error in lstat local path. %v\n", err)
		}
		return nil
	}
	if info.Mode() & (os.ModeSymlink | os.ModeDir) != 0 {
		(*reply).Error = userdrpc.BundleFileError_FBAD
		return nil
	}
	fileStartMtime := info.ModTime()
/*
	hdr := new(tar.Header)
	hdr.Size = info.Size()
	hdr.Name = args.Name
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Mode = 0644
*/
       // archive_file(w, r, info.Size())
	defer func() {
		if err := recover(); err != nil {
			bundleResetRPC(w, bundleInitialSize, reply, userdrpc.BundleFileError_UNKNOWN)
			return
		}
	} ()
//FIXME check for errors.
	_ = w.Sync()
	archiver.Archive(w, r, args.Name, info.Size(), info.ModTime().Unix())
/*
	tinfo, err := w.Stat()
	_, err = w.Seek(tinfo.Size(), 0)
	//err = tw.WriteHeader(hdr, info.Size())
	panicIfErr(err)
	for {
		num, err := r.Read(buffer[:])
		if (err != nil && err != io.EOF) || num < 0 {
			fmt.Fprintf(os.Stderr, "Userd: Failed to read. %v %v\n", err, num)
			panic("Error")
		}
		if err == io.EOF || num == 0 {
			break
		}
//FIXME error check this?
		hash.Write(buffer[0:num])

		w.Write(buffer[0:num])
	}
//FIXME check for errors.
	_ = w.Sync()
        archiver.Write_EOT(w, info.Size())
*/
//FIXME for now, simply reread the whole file for checksumming. This can be done in the archiver, but is tricky to pass back.
	r, err = os.Open(args.LocalPath)
	if err != nil {
		if os.IsNotExist(err) {
			(*reply).Error = userdrpc.BundleFileError_FNF
		} else if IsPermission(err) {
			(*reply).Error = userdrpc.BundleFileError_PERM
		} else {
			fmt.Fprintf(os.Stderr, "Got unknown error in open local path. %v\n", err)
		}
		return nil
	}
	defer r.Close()
	for {
		num, err := r.Read(buffer[:])
		if (err != nil && err != io.EOF) || num < 0 {
			fmt.Fprintf(os.Stderr, "Userd: Failed to read. %v %v\n", err, num)
			panic("Error")
		}
		if err == io.EOF || num == 0 {
			break
		}
//FIXME error check this?
		hash.Write(buffer[0:num])
	}
	err = w.Close()
	panicIfErr(err)
	info, err = os.Lstat(args.LocalPath)
	panicIfErr(err)
	fileEndMtime := info.ModTime()
	if fileEndMtime != fileStartMtime {
		bundleResetRPC(w, bundleInitialSize, reply, userdrpc.BundleFileError_CHANGED)
		return nil
	}
	(*reply).Retval = true
	(*reply).Error = userdrpc.BundleFileError_OK
	(*reply).Sha1 = fmt.Sprintf("%x", hash.Sum(nil))
	(*reply).Mtime = fileStartMtime
	return nil
}
