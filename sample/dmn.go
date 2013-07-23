package main

import (
	"flag"
	"go-daemon"
	"log"
	"os"
	"syscall"
)

const (
	pidFileName = "dmn.pid"
	logFileName = "dmn.log"

	fileMask = 0600
)

const (
	ret_OK = iota
	ret_ALREADYRUN
	ret_PIDFERROR
	ret_REBORNERROR
)

var (
	status = flag.Bool("status", false, "check status of the daemon")
	silent = flag.Bool("silent", false, "don't write in stdout")
)

func main() {
	flag.Parse()

	setupLogging()

	pidf := lockPidFile()

	err := daemon.Reborn(027, "./")
	if err != nil {
		log.Println("Reborn error:", err)
		os.Exit(ret_REBORNERROR)
	}

	log.Println("--- log ---")

	daemon.SetHandler(TermHandler, syscall.SIGTERM, syscall.SIGKILL)
	daemon.SetHandler(HupHandler, syscall.SIGHUP)
	daemon.SetHandler(Usr1Handler, syscall.SIGUSR1)

	err = daemon.ServeSignals()
	if err != nil {
		log.Println("Error:", err)
	}

	log.Println("--- end ---")

	pidf.Unlock()
}

func setupLogging() {
	if daemon.IsWasReborn() {
		file, _ := os.OpenFile(logFileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, fileMask)
		daemon.RedirectStream(os.Stdout, file)
		daemon.RedirectStream(os.Stderr, file)
		file.Close()
	} else {
		log.SetFlags(0)
		if *silent {
			file, _ := os.OpenFile(os.DevNull, os.O_WRONLY, fileMask)
			daemon.RedirectStream(os.Stdout, file)
			daemon.RedirectStream(os.Stderr, file)
			file.Close()
		}
	}
}

func lockPidFile() *daemon.PidFile {
	pidf, err := daemon.LockPidFile(pidFileName, fileMask)
	if err != nil {
		if err == daemon.ErrWouldBlock {
			log.Println("daemon copy is already running")
			os.Exit(ret_ALREADYRUN)
		} else {
			log.Println("pid file creation error:", err)
			os.Exit(ret_PIDFERROR)
		}
	}

	// unlock pid file, if deamon be reborn
	if !daemon.IsWasReborn() {
		pidf.Unlock()
	}

	// normal termination in case status checking
	if *status {
		os.Exit(ret_OK)
	}

	return pidf
}

func TermHandler(sig os.Signal) error {
	log.Println("SIGTERM:", sig)
	return daemon.ErrStop
}

func HupHandler(sig os.Signal) error {
	log.Println("SIGHUP:", sig)
	return nil
}

func Usr1Handler(sig os.Signal) error {
	log.Println("SIGUSR1:", sig)
	panic("fatal error")
}