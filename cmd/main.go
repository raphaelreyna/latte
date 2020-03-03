package main

import (
	"github.com/raphaelreyna/latte/server"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	errLog := log.New(os.Stderr, "ERROR: ", log.Lshortfile|log.LstdFlags)
	infoLog := log.New(os.Stdout, "INFO: ", log.Lshortfile|log.LstdFlags)

	if _, err := exec.LookPath("pdflatex"); err != nil {
		errLog.Fatal("pdflatex binary not found in your $PATH")
	}

	db, err := newDB()
	if err != nil {
		errLog.Fatal(err)
	}

	s, err := server.NewServer(os.Getenv("LATTE_ROOT"), db, errLog, infoLog)
	if err != nil {
		errLog.Fatal(err)
	}
	errLog.Fatal(http.ListenAndServe(":27182", s))
}
