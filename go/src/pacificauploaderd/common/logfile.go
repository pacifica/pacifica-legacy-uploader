package common

import (
	"fmt"
	"log"
	"pacifica/redirectstd"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

//Go's logger sucks!
//You can't SetOutput on any logger except the default one.
//You can't get the default logger so that you can call Output on it
//You can't set the default logger to your own new logger so you can have a handle for Output
//If you don't use Output directly, you loose all stack tracing abilities, which means your just as good off just using a raw text file and forgetting about the logger all together.

//For now, call the logger implementation and loose all stack tracing. This sucks, but I don't care to reimplement all of that right now.

func Dprint(v ...interface{}) {
	if Devel {
		log.Print(v...)
	}
}

func Dprintf(format string, v ...interface{}) {
	if Devel {
		log.Printf(format, v...)
	}
}

func Dprintln(v ...interface{}) {
	if Devel {
		log.Println(v...)
	}
}

const _MAX_LOG_SIZE int64 = 1e8 //100 MB

func setupLogger(dir string) {
	path := filepath.Join(dir, _LOG_FILENAME)

	// SetFlags customizes the output into the log file.
	// LstdFlags is Ldate | Ltime
	// Lshortfile
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rotate(path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		log.Printf("Failed to set up log file %s, error %+v", path, err)
		return
	}
	log.Printf("Setting log output to %s", path)

	redirectstd.RedirectStdErr(f)
	log.SetOutput(f)
}

func rotate(path string) {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		//If the file does not exist, just skip.
		return
	}

	if err != nil {
		log.Printf("Failed to stat %s", path)
		log.Printf("Cannot rotate log file %s, skipping rotate", path)
		return
	}

	if fi.Size() >= _MAX_LOG_SIZE {
		ext := filepath.Ext(path)
		pattern := path[0:len(path)-len(ext)] + ".*" + ext //e.g. path.0.log
		fmt.Printf("pattern is %s\n", pattern)
		files, err := filepath.Glob(pattern)
		if err != nil {
			log.Printf("Could not get files for %s, error %+v", pattern, err)
			log.Printf("Cannot rotate log file %s, skipping rotate", path)
			return
		}

		fileBase := path[0 : len(path)-len(ext)]

		//TODO - this sort does not work with file names with numerics
		//greater than 9.
		sort.Strings(files)

		//Debug...
		for _, v := range files {
			fmt.Printf("%s\n", v)
		}

		//Rotate old log files, e.g. those ending with the form *.0.ext
		for i := len(files) - 1; i >= 0; i-- {
			file := files[i]
			middleExt := "." + strconv.Itoa(i+1)
			newExt := middleExt + ext
			newFilename := fileBase + newExt
			fmt.Printf("Renaming %s to %s\n", file, newFilename)
			err := os.Rename(file, newFilename)
			if err != nil {
				log.Printf("Error renaming %s to %s, %+v", file, newFilename)
				log.Printf("Skipping rotate")
				return
			}
		}

		//Rename current path file to path.0.ext
		newFilename := fileBase + "." + strconv.Itoa(0) + ext
		fmt.Printf("Renaming %s to %s\n", path, newFilename)
		err = os.Rename(path, newFilename)
		if err != nil {
			log.Printf("Error renaming %s to %s, %+v", path, newFilename)
			log.Printf("Continue without rotate")
			return
		}
	}
}
