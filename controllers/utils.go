package controllers

import (
	"bytes"
	"fmt"
	"net/http"
)

//
// Helper functions to check and remove string from a slice of strings.
//
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func SendLBToSubscribe(uri string, method string, data []byte) error {

	client := &http.Client{}
	req, _ := http.NewRequest(method, uri, bytes.NewBuffer(data))
	req.Header.Set("Content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "http request failed")
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 == 2 {
		log.Info("Send loadBalance configuration to subscribe successfully")
	} else {
		return fmt.Errorf("unexpected status-code returned from subscribe %v", resp.StatusCode)
	}

	return nil
}
