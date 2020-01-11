package main

import (
	"github.com/raphaelreyna/latte/server"
	"log"
	"os/exec"
)

func main() {
	// Make sure pdflatex is installed on the host
	if !HasPDFLatex() {
		log.Fatal("pdflatex binary not found in your $PATH")
	}
	// Get config
	c := server.DefaultedConfig()
	log.Fatal(server.ListenAndServe(c))
}

func HasPDFLatex() bool {
	_, err := exec.LookPath("pdflatex")
	return err == nil
}
