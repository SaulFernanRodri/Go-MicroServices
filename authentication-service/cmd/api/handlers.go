package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		if err := app.errorJSON(w, err, http.StatusBadRequest); err != nil {
			log.Println("Error writing JSON response:", err)
			return
		}
		return
	}

	// validate the user against the database
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		if err := app.errorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized); err != nil {
			log.Println("Error writing JSON response:", err)
			return
		}

		return
	}

	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		if err := app.errorJSON(w, errors.New("invalid credentials"), http.StatusUnauthorized); err != nil {
			log.Println("Error writing JSON response:", err)
			return
		}
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", user.Email),
		Data:    user,
	}

	if err := app.writeJSON(w, http.StatusOK, payload); err != nil {
		log.Println("Error writing JSON response:", err)
	}
}
