package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

type AuthorizedClient struct {
	Id     string
	Secret string
}

func NewAuthorizedClient(id string, secret string) AuthorizedClient {
	client := AuthorizedClient{
		Id:     id,
		Secret: secret,
	}

	path := getClientIDPath()
	if fileExists(path) {
		values, err := ReadJsonFile(path)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		content := values.(map[string]interface{})
		installed := content["installed"].(map[string]interface{})
		client.Id = installed["client_id"].(string)
		client.Secret = installed["client_secret"].(string)
	}

	return client
}

func getClientIDPath() string {

	path := os.Getenv("GDRIVE_CLIENT_ID_PATH")
	if len(path) != 0 {
		return path
	} else {
		return filepath.Join(GetConfigDir(), "client_id.json")
	}
}
