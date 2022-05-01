package log

import (
	"log"
	"os"
)

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
	FatalLogger   *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stderr, "INFO:  ", log.Lmsgprefix)
	WarningLogger = log.New(os.Stderr, "WARN:  ", log.Lmsgprefix)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Lmsgprefix)
	FatalLogger = log.New(os.Stderr, "FATAL: ", log.Lmsgprefix)
}
