package main

import (
	"github.com/raphaelreyna/latte/internal/server"
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

	db, err := newDB(infoLog)
	if err != nil {
		errLog.Fatal(err)
	}

	root := os.Getenv("LATTE_ROOT")
	if root == "" {
		root, err = os.UserCacheDir()
		if err != nil {
			errLog.Fatal("error creating root cache directory: %v", err)
		}
	}
	s, err := server.NewServer(root, db, errLog, infoLog)
	if err != nil {
		errLog.Fatal(err)
	}
	infoLog.Printf("root cache directory: %s", root)
	port := os.Getenv("LATTE_PORT")
	if port == "" {
		port = "27182"
	}
	infoLog.Printf("listening for HTTP traffic on port: %s ...", port)
	errLog.Fatal(http.ListenAndServe(":"+port, s))
}
