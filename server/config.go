package server

import (
	"os"
	"path/filepath"
)

type ServerConf struct {
	RootDir   string
	Port      string
	DbType    string
	DbConnStr string
}

func GrabConfigFromEnv() *ServerConf {
	var c ServerConf
	// Grab root directory
	c.RootDir = os.Getenv("LATTE_ROOT_DIR")
	// Grab port
	c.Port = os.Getenv("LATTE_PORT")
	if c.Port != "" {
		c.Port = ":" + c.Port
	}
	// Grab database type (PostgreSQL, MySQL or SQLite)
	c.RootDir = os.Getenv("LATTE_DBTYPE")
	// Grab DbConnStr
	c.RootDir = os.Getenv("LATTE_DBCONNSTR")
	return &c
}

func DefaultedConfig() *ServerConf {
	c := GrabConfigFromEnv()
	// Check RootDir
	if c.RootDir == "" {
		c.RootDir = filepath.Join(
			os.TempDir(),
			"com.raphaelreyna.latte",
		)
	}
	// Check Port
	if c.Port == "" {
		c.Port = ":27182"
	}
	// Check DbType
	if c.DbType == "" {
		c.DbType = "sqlite"
	}
	// Check conn string
	if c.DbConnStr == "" {
		c.DbConnStr = ":memory:"
	}
	return c
}
