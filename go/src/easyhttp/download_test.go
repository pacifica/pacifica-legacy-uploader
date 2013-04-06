package easyhttp

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const (
	hostname = "127.0.0.1"
	port     = 8999
)

func TestDownloadAsync(t *testing.T) {
	//TODO this was an experiment to have a completely 
	//self-contained unit test.  It starts creates a file,
	//starts a webserver and downloads the file.  The first
	//two steps don't seem to work at the moment so we'll just
	//use http://google.com until we get around to fixing it.
	//fmt.Printf("Entering TestDownloadAsync\n")
	//defer fmt.Printf("Leaving TestDownloadAsync\n")

	//path, err := createFile()
	//if err != nil {
	//	t.Fatalf("createFile failed %v\n", err)
	//}

	//serveFile(path, hostname, port)

	//url := fmt.Sprintf("http://%s:%d/%s", hostname, port, filepath.Base(path))
	//saveAs := filepath.Join(os.TempDir(), filepath.Base(path)+".downloaded")

	url := "http://google.com"
	dir := filepath.Join(os.TempDir(), "download_test")
	saveAs := filepath.Join(dir, "google.com.txt")

	fmt.Printf("url is %s\n", url)
	fmt.Printf("saveAs is %s\n", saveAs)

	result := DownloadAsync(url, saveAs)
	dr := <-result

	if dr.Err != nil {
		t.Fatalf("dr.Err %v\n", dr.Err)
	}

	if dr.Path != saveAs {
		t.Fatalf("dr.Path (%s) != saveAs (%s)\n", dr.Path, saveAs)
	}

	/*err := os.Remove(saveAs)
	if err != nil {
		//fmt.Printf("Could not remove %s\n, %v", saveAs, err)
		t.Fatalf("Could not remove %s\n, %v", saveAs, err)
	}

	err = os.Remove(dir)
	if err != nil {
		//fmt.Printf("Could not remove %s\n, %v", saveAs, err)
		t.Fatalf("Could not remove %s\n, %v", saveAs, err)
	}*/
}

func createFile() (path string, err error) {
	fmt.Printf("Entering createFile\n")
	defer fmt.Printf("Leaving createFile\n")

	t, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer t.Close()
	io.WriteString(t, "Hello download")
	return t.Name(), nil
}

func serveFile(path, host string, port int) {
	fmt.Printf("Entering serveFile\n")
	defer fmt.Printf("Leaving serveFile\n")

	parentDir := filepath.Dir(path)
	fmt.Printf("parent dir is %s\n", parentDir)
	http.Handle("/",
		http.FileServer(http.Dir(filepath.Dir(parentDir))))
	go http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil)
}
