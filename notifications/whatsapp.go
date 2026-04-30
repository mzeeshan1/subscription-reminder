package notifications

import (
	twilio "github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

type whatsappSender struct {
	client     *twilio.RestClient
	fromNumber string
}

func newWhatsApp(accountSID, authToken, fromNumber string) *whatsappSender {
	if accountSID == "" {
		return nil
	}
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})
	return &whatsappSender{client: client, fromNumber: fromNumber}
}

func (w *whatsappSender) send(toNumber, message string) error {
	to := "whatsapp:" + toNumber
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(w.fromNumber)
	params.SetBody(message)
	_, err := w.client.Api.CreateMessage(params)
	return err
}
