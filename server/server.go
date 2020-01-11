package server

import (
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"net/http"
	"os"
)

type Handler struct {
	HandlerFunc func(db *gorm.DB, w http.ResponseWriter, r *http.Request)
	DB          *gorm.DB
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.HandlerFunc(h.DB, w, r)
}

func EnsureRootDir(c *ServerConf) error {
	// Check if rootdir exists
	_, err := os.Stat(c.RootDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(c.RootDir, 0755)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func ListenAndServe(c *ServerConf) error {
	err := EnsureRootDir(c)
	if err != nil {
		return err
	}
	router := mux.NewRouter()
	router.Handle("/generate", &Handler{
		HandlerFunc: handleGenerate,
	})
	return http.ListenAndServe(c.Port, router)
}
