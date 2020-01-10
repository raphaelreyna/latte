package server

import (
	"encoding/base64"
	"os"
)

func SaveEnvToDisk(uid string, resources map[string]string) error {
	// Create directory for resources
	err := os.Mkdir(uid, 0755)
	if err != nil {
		return err
	}
	err = os.Chdir(uid)
	if err != nil {
		return err
	}
	// Loop over resources and save them
	for name, file := range resources {
		err = SaveFile(name, file)
		if err != nil {
			return err
		}
	}
	os.Chdir("..")
	return nil
}

func SaveFile(fileName, encodedFile string) error {
	// Grab template bytes
	fBytes, err := base64.StdEncoding.DecodeString(encodedFile)
	if err != nil {
		return err
	}
	// Create new template.tex file
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	// Write the template bytes to the template file
	_, err = file.Write(fBytes)
	if err != nil {
		return err
	}
	err = file.Sync()
	if err != nil {
		return err
	}
	file.Close()
	return nil
}
