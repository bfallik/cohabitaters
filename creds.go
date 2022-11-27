package cohabitaters

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func ConfigFromJSONFile(path string, scopes ...string) (*oauth2.Config, error) {

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading '%s': %w", path, err)
	}

	return google.ConfigFromJSON(b, scopes...)
}
