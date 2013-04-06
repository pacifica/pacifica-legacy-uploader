package easyhttp

import (
	"errors"
	"path/filepath"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DownloadResult struct {
	Err  error
	Path string
}

func DownloadAsync(httpUrl, savePath string) chan DownloadResult {
	result := make(chan DownloadResult, 1)
	go func() {
		c := new(http.Client)
		r, err := c.Get(httpUrl)
		if err != nil {
			result <- DownloadResult{Err: err}
			return
		} else if r.StatusCode != 200 {
			errMsg := fmt.Sprintf("Download failed with status code %d.", r.StatusCode)
			result <- DownloadResult{Err: errors.New(errMsg)}
			return
		}

		parent := filepath.Dir(savePath)
		if parent == "" || parent == "." {
			errMsg := fmt.Sprintf("Could not save file, %v is not a valid directory.", savePath)
			result <- DownloadResult{Err: errors.New(errMsg)}
			return
		}

		err = os.MkdirAll(parent, 0777)
		if err != nil {
			errMsg := fmt.Sprintf("Could not make directory %v, %v.", savePath, err)
			result <- DownloadResult{Err: errors.New(errMsg)}
			return
		}

		dst, err := os.Create(savePath)
		if err != nil {
			result <- DownloadResult{Err: err}
			return
		}

		_, err = io.Copy(dst, r.Body)
		if err != nil {
			result <- DownloadResult{Err: err}
			return
		}

		//Close these files before sending DownloadResult through
		//channel so that the reciever can immediately start using it.
		//Closing these using defer causes the file to be closed after it 
		//returned through the channel.
		dst.Close()
		r.Body.Close()

		result <- DownloadResult{Path: savePath}
	}()
	return result
}
