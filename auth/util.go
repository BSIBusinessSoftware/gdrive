package auth

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

func mkdir(path string) error {
	dir := filepath.Dir(path)
	if fileExists(dir) {
		return nil
	}
	return os.Mkdir(dir, 0700)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func GetConfigDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "gdrive")
}

func ReadJsonFile(path string) (interface{}, error) {

	content, _ := ioutil.ReadFile(path)
	var data interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
