package archiver

import (
	"io"
	"os"
	"fmt"
	"testing"
	"crypto/sha1"
	"archive/tar"
)

const (
	BUFFSIZE int = 1024
)

func cleanup() {
	os.Remove("test.tar")
	os.Remove("test.file")
}

func Test_BasicTar(t *testing.T) {
	sha1sum := "a1578f3c9b7798bb6fdfa523315ae2475fe73f52"
	testdata := "This is test data to be tared up."
	hash := sha1.New()
	var buffer [BUFFSIZE]byte
	w, err := os.Create("test.tar")
	if err != nil {
        	t.Error("Failed to open test tar. %v", err)
	} else {
		f, err := os.Create("test.file")
		if err != nil {
			t.Error("Failed to open test tar.")
		} else {
			data := []byte("This is test data to be tared up.")
			f.Write(data)
			f.Seek(0, os.SEEK_SET)
			i := Archive(w, f, "test.filename", int64(len(testdata)), 0)
			if i != 0 {
				t.Error("Archive returned an error")
			} else {
				w.Seek(0, os.SEEK_SET)
				tr := tar.NewReader(w)
				for {
					hdr, err := tr.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Error("Userd: Failed to read tar.")
						cleanup()
						return
					}
					if hdr.Name == "test.filename" {
						for {
							num, err := tr.Read(buffer[:])
							if (err != nil && err != io.EOF) || num < 0 {
								t.Error("Userd: Failed to read. %v %v\n", err, num)
								cleanup()
								return
							}
							if err == io.EOF || num == 0 {
								break
							}
							hash.Write(buffer[0:num])
						}
					}
				}
				tmpsum := fmt.Sprintf("%x", hash.Sum(nil))
				if sha1sum == tmpsum {
					t.Log("one test passed.")
				} else {
					t.Error("Sha1 sums don't match.", sha1sum, tmpsum)
				}
			}
		}
	}
	cleanup()
}
