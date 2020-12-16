package utils

import (
	"log"
	"os"
	"path/filepath"
)

func GetRootPath() (dir string) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Panicln("发生错误", err.Error())
	}
	return
}
