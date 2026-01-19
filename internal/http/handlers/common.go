package handlers

import (
	"encoding/json"
	"io"
	"net/http"
)

func DecodeJSON(r *http.Request, target any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, target)
}
