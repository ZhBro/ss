package utils

import (
	"io"
	"os"
)

func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func AppendToFile(context string, filePath string) error {
	f, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		return err
	} else {
		n, err := f.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}
		_, err = f.WriteAt([]byte(context), n)
		if err != nil {
			return err
		}
	}
	return nil
}
