package ylog

import (
	"YTools/ycomm"
	"fmt"
	"log"
	"os"
)

var logger = log.New(os.Stderr, "", log.Lshortfile|log.LstdFlags)

func Yprint(a ...interface{}) {
	//先打印输出
	fmt.Println(a)
}

func Logf(f string, v ...interface{}) {
	if ycomm.Debug {
		logger.Output(2, fmt.Sprintf(f, v...))
	}
}

type logHelper struct {
	prefix string
}

func (l *logHelper) Write(p []byte) (n int, err error) {
	if ycomm.Debug {
		logger.Printf("%s%s\n", l.prefix, p)
		return len(p), nil
	}
	return len(p), nil
}

func newLogHelper(prefix string) *logHelper {
	return &logHelper{prefix}
}
