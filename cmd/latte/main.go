package main

import (
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/raphaelreyna/latte/internal/server"
	"github.com/rs/zerolog/log"
)

const (
	// If cache sizes is not provided by environment, default to 15 for both
	defaultTCS = 15
	defaultRCS = 15
)

var db server.DB

func main() {
	var err error

	// Check for pdfLaTeX (pdfTex will do in a pinch)
	cmd := "pdflatex"
	if _, err := exec.LookPath(cmd); err != nil {
		log.Error().Err(err).
			Msg("pdflatex not found, checking for pdftex")

		if _, err := exec.LookPath("pdftex"); err != nil {
			log.Fatal().Err(err).
				Msg("pdftex not found")
		}

		log.Info().
			Msg("found pdftex binary; falling back to using pdftex instead of pdflatex")

		cmd = "pdftex"
	}

	// If user provides a directory path or a tex file, then run as cli tool and not as http server
	if len(os.Args) > 1 {
		if os.Args[1] != "server" {
			cli()
			os.Exit(0)
		}
	}
	root := os.Getenv("LATTE_ROOT")
	if root == "" {
		root, err = os.UserCacheDir()
		if err != nil {
			log.Fatal().Err(err).
				Msg("error creating root cache directory")
		}
	}
	log.Info().Str("path", root).
		Msg("root cache directory")

	tCacheSize := os.Getenv("LATTE_TMPL_CACHE_SIZE")
	tcs, err := strconv.Atoi(tCacheSize)
	if err != nil {
		log.Warn().
			Int("default", defaultTCS).
			Msg("couldn't pull templates cache size from environment, using default")
		tcs = defaultTCS
	}
	s, err := server.NewServer(root, cmd, db, tcs)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "27182"
	}
	log.Info().
		Str("port", port).Msg("listening for HTTP traffic")

	err = http.ListenAndServe(":"+port,
		handlers.CORS(
			handlers.AllowedHeaders([]string{
				"X-Requested-With", "Content-Type", "Authorization", "Access-Control-Allow-Origin",
			}),
			handlers.AllowedMethods([]string{
				"GET", "POST", "PUT", "HEAD", "OPTIONS",
			}),
			handlers.AllowedOrigins([]string{"*"}),
		)(s),
	)

	if err != nil {
		log.Fatal().Err(err).Send()
	}
}
