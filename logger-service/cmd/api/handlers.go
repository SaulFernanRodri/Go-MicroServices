package main

import (
	"log"
	"log-service/data"
	"net/http"
)

type JSONPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	// read json into var
	var requestPayload JSONPayload
	_ = app.readJSON(w, r, &requestPayload)

	// insert data
	event := data.LogEntry{
		Name: requestPayload.Name,
		Data: requestPayload.Data,
	}

	err := app.Models.LogEntry.Insert(event)
	if err != nil {
		if err := app.errorJSON(w, err); err != nil {
			log.Println(err)
			return
		}
		return
	}

	resp := jsonResponse{
		Error:   false,
		Message: "logged",
	}

	if err := app.writeJSON(w, http.StatusAccepted, resp); err != nil {
		log.Println(err)
		return
	}
}
