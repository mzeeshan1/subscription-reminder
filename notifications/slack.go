package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type slackSender struct{}

func (s *slackSender) send(webhookURL, text string) error {
	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}
