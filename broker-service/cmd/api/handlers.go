package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/rpc"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RequestPayload describes the JSON that this service accepts as an HTTP Post request
type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

// MailPayload is the embedded type (in RequestPayload) that describes an email message to be sent
type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// AuthPayload is the embedded type (in RequestPayload) that describes an authentication request
type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LogPayload is the embedded type (in RequestPayload) that describes a request to log something
type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// Broker is a test handler, just to make sure we can hit the broker from a web client
func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	if err := app.writeJSON(w, http.StatusOK, payload); err != nil {
		return
	}
}

// HandleSubmission is the main point of entry into the broker. It accepts a JSON
// payload and performs an action based on the value of "action" in that JSON.
func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	case "log.http":
		app.logItem(w, requestPayload.Log)
	case "log":
		app.logItemViaRPC(w, requestPayload.Log)
	case "log.rabbit":
		app.logEventViaRabbit(w, requestPayload.Log)
	case "mail":
		app.sendMail(w, requestPayload.Mail)
	default:
		if err := app.errorJSON(w, errors.New("unknown action")); err != nil {
			log.Println(err)
		}
	}
}

// logItem logs an item by making an HTTP Post request with a JSON payload, to the logger microservice
func (app *Config) logItem(w http.ResponseWriter, entry LogPayload) {
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Println("Error closing response body", err)
		}
	}()

	if response.StatusCode != http.StatusAccepted {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"

	if err := app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		return
	}

}

// authenticate calls the authentication microservice and sends back the appropriate response
func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Println("Error closing response body", err)
		}
	}()
	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		if err := app.errorJSON(w, errors.New("invalid credentials")); err != nil {
			log.Println(err)
			return
		}
		return
	} else if response.StatusCode != http.StatusBadRequest {
		if err := app.errorJSON(w, errors.New("bad request")); err != nil {
			log.Println(err)
			return
		}
		return

	} else if response.StatusCode != http.StatusAccepted {
		if err := app.errorJSON(w, errors.New("error calling auth service")); err != nil {
			log.Println(err)
			return
		}
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	if jsonFromService.Error {
		if err := app.errorJSON(w, err, http.StatusUnauthorized); err != nil {
			log.Println(err)
			return
		}
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated!"
	payload.Data = jsonFromService.Data

	if err := app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		return
	}
}

// sendMail sends email by calling the mail microservice
func (app *Config) sendMail(w http.ResponseWriter, msg MailPayload) {
	jsonData, _ := json.MarshalIndent(msg, "", "\t")

	// call the mail service
	mailServiceURL := "http://mailer-service/send"

	// post to mail service
	request, err := http.NewRequest("POST", mailServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			log.Println("Error closing response body", err)
		}
	}()
	// make sure we get back the right status code
	if response.StatusCode != http.StatusAccepted {
		if err := app.errorJSON(w, errors.New("error calling mail service")); err != nil {
			log.Println(err)
			return
		}
		return
	}

	// send back json
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Message sent to " + msg.To

	if err := app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		return
	}

}

// logEventViaRabbit logs an event using the logger-service. It makes the call by pushing the data to RabbitMQ.
func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged via RabbitMQ"

	if err := app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		return
	}
}

// pushToQueue pushes a message into RabbitMQ
func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	j, err := json.MarshalIndent(&payload, "", "\t")
	if err != nil {
		return err
	}

	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}

type RPCPayload struct {
	Name string
	Data string
}

// logItemViaRPC logs an item by making an RPC call to the logger microservice
func (app *Config) logItemViaRPC(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	rpcPayload := RPCPayload(l)

	var result string
	err = client.Call("RPCServer.LogInfo", &rpcPayload, &result)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}

	if err := app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		return
	}
}

func (app *Config) LogViaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	conn, err := grpc.NewClient("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Println("Error closing connection", err)
		}
	}()

	c := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"

	if err := app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		return
	}
}
