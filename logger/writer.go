package logger

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/SSSOC-CAN/laniakea/utils"
)

type moddedFileWriter struct {
	File         *os.File
	maxFileSize  uint32 // MB
	maxFiles     uint16
	fileNameRoot string // must include path if not in current directory
	fileExt      string
	pathToFile   string
}

// Write Implements the io.Writer interface
func (w *moddedFileWriter) Write(p []byte) (n int, err error) {
	stat, err := w.File.Stat()
	if err != nil {
		return 0, err
	}
	// Check if maximum file size if exceeded
	if stat.Size() >= int64(w.maxFileSize) || stat.Size()+int64(len(p)) >= int64(w.maxFileSize) {
		// get current file name number
		r, err := regexp.Compile(fmt.Sprintf("%s([0-9]+)", w.fileNameRoot)) // may need file extension
		if err != nil {
			return 0, err
		}
		matches := r.FindStringSubmatch(stat.Name())
		var (
			fileNum int64
		)
		if len(matches) > 1 {
			fileNum, err = strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				return 0, err
			}
		}
		// Close current file and delete new file if it already exists
		w.File.Close()
		var newFileName string
		if fileNum >= int64(w.maxFiles-1) {
			newFileName = fmt.Sprintf("%s.%s", w.fileNameRoot, w.fileExt)
		} else {
			newFileName = fmt.Sprintf("%s%v.%s", w.fileNameRoot, fileNum+int64(1), w.fileExt)
		}
		if utils.FileExists(fmt.Sprintf("%s/%s", w.pathToFile, newFileName)) {
			err = os.Remove(fmt.Sprintf("%s/%s", w.pathToFile, newFileName))
			if err != nil {
				return 0, err
			}
		}
		newFile, err := os.OpenFile(fmt.Sprintf("%s/%s", w.pathToFile, newFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0775)
		if err != nil {
			return 0, err
		}
		w.File = newFile
	}
	return w.File.Write(p)
}
