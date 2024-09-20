package main

import (
	"broker/event"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   Mail        `json:"mail,omitempty"`
}

type Mail struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) Broker(rw http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit me",
	}

	_ = app.writeJSON(rw, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(rw http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(rw, r, &requestPayload)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}
	fmt.Printf("Received payload: %+v\n", requestPayload)
	fmt.Printf("Received payload: %+v\n", requestPayload.Auth)
	fmt.Printf("Received payload: %+v\n", requestPayload.Log)
	switch requestPayload.Action {
	case "auth":
		app.authenticate(rw, requestPayload.Auth)
	case "log":
		app.logViaRabbitMQ(rw, requestPayload.Log)
	case "mail":
		app.sendMail(rw, requestPayload.Mail)
	default:
		app.errorJSON(rw, errors.New("unknow action"))

	}
}

func (app *Config) logItem(rw http.ResponseWriter, entry LogPayload) {
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(rw, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	fmt.Printf("Received requesttttt: %+v\n", request)
	fmt.Printf("Received esponseeeee: %+v\n", response)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(rw, fmt.Errorf("unexpected status code: %d", response.StatusCode), http.StatusInternalServerError)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged!"

	app.writeJSON(rw, http.StatusAccepted, payload)
}

func (app *Config) authenticate(rw http.ResponseWriter, a AuthPayload) {
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(rw, err)
		return
	}
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}
	fmt.Printf("Received requesttttt: %+v\n", request)
	fmt.Printf("Received esponseeeee: %+v\n", response)
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(rw, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.errorJSON(rw, fmt.Errorf("unexpected status code: %d", response.StatusCode), http.StatusInternalServerError)
		return
	}

	var jsonFromService jsonResponse

	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(rw, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authed!"
	payload.Data = jsonFromService.Data

	app.writeJSON(rw, http.StatusAccepted, payload)
}

func (app *Config) sendMail(rw http.ResponseWriter, mail Mail) {
	jsonData, _ := json.MarshalIndent(mail, "", "\t")

	mailServiceURL := "http://mail-service/send"

	request, err := http.NewRequest("POST", mailServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(rw, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}
	fmt.Printf("Received requesttttt maill: %+v\n", request)
	fmt.Printf("Received esponseeeee maiill: %+v\n", response)
	bodyBytes, _ := io.ReadAll(response.Body)
	fmt.Printf("Response Body: %s\n", string(bodyBytes))
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(rw, fmt.Errorf("unexpected mail status code: %d", response.StatusCode), http.StatusInternalServerError)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Sent! " + mail.To

	app.writeJSON(rw, http.StatusAccepted, payload)
}

func (app *Config) logViaRabbitMQ(rw http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		app.errorJSON(rw, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged via Rabbitmq"
	app.writeJSON(rw, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}
	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	j, _ := json.MarshalIndent(&payload, "", "\t")
	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}
