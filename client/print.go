package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func SendToPrinter(export []Component, url string) error {
	j, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}

	if url == "" {
		return fmt.Errorf("print server URL not set")
	}

	resp, err := http.Post(url+"/print-receipt", "application/json", bytes.NewBuffer(j))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("print failed: %s", body)
	}
	return nil
}
