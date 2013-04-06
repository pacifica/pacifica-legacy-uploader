// Creates an environment to test pacificauploaderd against.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
)

const (
	fileCount   int64 = 42e3
	maxFileSize int64 = 1e3
	subDirCount       = 10
	baseDir           = "C:\\testbed"
	//subDirs = []string{"Instrument A/Experiment B",
	//	"Raw Data",
	//	"Papers",
	//	"Intrument A/Input"}
)

var dirs []string

func main() {
	setupDirs()
	setupFiles()
}

func setupDirs() {
	dirs = make([]string, subDirCount+1)
	//baseDir should not exist before this program runs.
	err := os.Mkdir(baseDir, 777)
	if err != nil {
		log.Fatalf("%v", err)
	}
	dirs[0] = baseDir
	for i := 1; i < len(dirs); i++ {
		//Get random number between 0 and i - 1
		parentDirIndex := rand.Intn(i)

		//Get a random parent directory.  These paths will grow deeper (randomly) as 
		//subDirCount increases.
		parentDir := dirs[parentDirIndex]

		//Create sub-directory
		subDir := filepath.Join(parentDir, fmt.Sprintf("sub%d", i))
		err = os.MkdirAll(subDir, 777)
		if err != nil {
			log.Fatalf("%v", err)
		}

		//Add sub-directory to list
		dirs[i] = subDir

		log.Printf("%s", subDir)
	}
}

func setupFiles() {
	for i := int64(0); i < fileCount; i++ {
		//Choose a random parent directory for file
		parentDirIndex := rand.Intn(len(dirs))
		parentDir := dirs[parentDirIndex]
		
		//Create file with random name
		f, err := ioutil.TempFile(parentDir, "")
		if err != nil {
			log.Fatalf("%v", err)
		}
		log.Printf("%s", f.Name())		
		writeFile(f, maxFileSize)		
		f.Close()
	}
}

func writeFile(f *os.File, max int64) {
	//Total length of file including max
	length := rand.Int63n(max + 1)

	//Load a buffer with garbage
	buff := make([]byte, 512)
	for i := range buff {
		buff[i] = byte(rand.Intn(256))
	}

	//Repeatedly write buff until length is reached
	written := int64(0)
	for {
		if written >= length {
			break
		}

		remaining := length - written
		var n int
		if int64(len(buff)) > remaining {
			n, _ = f.Write(buff[0:remaining])
		} else {
			n, _ = f.Write(buff)
		}
		written += int64(n)
	}

	if written != length {
		panic("written != length")
	}
}
