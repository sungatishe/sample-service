package main

import "net/http"

func (app *Config) SendMail(rw http.ResponseWriter, r *http.Request) {
	type mailMessage struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}

	var requestPayload mailMessage

	err := app.readJSON(rw, r, &requestPayload)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}

	msg := Message{
		From:    requestPayload.From,
		To:      requestPayload.To,
		Subject: requestPayload.Subject,
		Data:    requestPayload.Message,
	}

	err = app.Mailer.SendSMTPMessage(msg)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "sent to " + requestPayload.To,
	}

	app.writeJSON(rw, http.StatusAccepted, payload)
}
