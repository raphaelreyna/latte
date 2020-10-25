package main

import (
	"github.com/gorilla/handlers"
	"github.com/raphaelreyna/latte/internal/server"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
)

const (
	// If cache sizes is not provided by environment, default to 15 for both
	defaultTCS = 15
	defaultRCS = 15
)

var db server.DB

func main() {
	var err error
	errLog := log.New(os.Stderr, "ERROR: ", log.Lshortfile|log.LstdFlags)
	infoLog := log.New(os.Stdout, "INFO: ", log.Lshortfile|log.LstdFlags)

	// Check for pdfLaTeX (pdfTex will do in a pinch)
	cmd := "pdflatex"
	if _, err := exec.LookPath(cmd); err != nil {
		errLog.Printf("error while searching checking pdflatex binary: %v\n\tchecking for pdftex binary", err)
		if _, err := exec.LookPath("pdftex"); err != nil {
			errLog.Fatal("neither pdflatex nor pdftex binary found in your $PATH")
		}
		infoLog.Printf("found pdftex binary; falling back to using pdftex instead of pdflatex")
		cmd = "pdftex"
	}

	// If user provides a directory path or a tex file, then run as cli tool and not as http server
	if len(os.Args) > 1 {
		if os.Args[1] != "server" {
			cli(cmd, errLog, infoLog)
			os.Exit(0)
		}
	}
	root := os.Getenv("LATTE_ROOT")
	if root == "" {
		root, err = os.UserCacheDir()
		if err != nil {
			errLog.Fatalf("error creating root cache directory: %v", err)
		}
	}
	infoLog.Printf("root cache directory: %s", root)

	tCacheSize := os.Getenv("LATTE_TMPL_CACHE_SIZE")
	tcs, err := strconv.Atoi(tCacheSize)
	if err != nil {
		infoLog.Printf("couldn't pull templates cache size from environment: defaulting to %d", defaultTCS)
		tcs = defaultTCS
	}
	s, err := server.NewServer(root, cmd, db, errLog, infoLog, tcs)
	if err != nil {
		errLog.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "27182"
	}
	infoLog.Printf("listening for HTTP traffic on port: %s ...", port)
	errLog.Fatal(http.ListenAndServe(":"+port, handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization", "Access-Control-Allow-Origin"}), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}), handlers.AllowedOrigins([]string{"*"}))(s)))
}
