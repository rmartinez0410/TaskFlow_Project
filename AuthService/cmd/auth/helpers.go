package main

import (
	"auth/internal/data"
	"auth/internal/validator"
	"encoding/json"
	"net/http"

	"github.com/nats-io/nats.go"
)

func (app *application) readJSON(msg *nats.Msg, dst any, f func(v *validator.Validator)) bool {
	err := json.Unmarshal(msg.Data, dst)
	if err != nil {
		app.sendUnprocessableEntityResponse(msg)
		return false
	}

	v := validator.New()
	if f(v); !v.Valid() {
		app.sendErrorResponse(msg, http.StatusUnprocessableEntity, v.Errors)
		return false
	}
	return true
}

func (app *application) sendErrorResponse(msg *nats.Msg, status int, message any) {
	response := &data.Response{
		StatusCode: status,
		Data:       message,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		app.logger.Error("failed to marshal error response", "error", err, "original", message)

		err := msg.Respond([]byte(`{"status":500,"error":"internal server error"}`))
		if err != nil {
			app.logger.Error("failed to send fallback response", "error", err)
			return
		}
	}

	err = msg.Respond(responseData)
	if err != nil {
		app.logger.Error("failed to send error response", "error", err)
	}
}

func (app *application) sendInternalServerErrorResponse(msg *nats.Msg) {
	app.sendErrorResponse(msg, http.StatusInternalServerError, "internal server error")
}

func (app *application) sendUnprocessableEntityResponse(msg *nats.Msg) {
	app.sendErrorResponse(msg, http.StatusUnprocessableEntity, "unprocessable entity")
}

func (app *application) sendSuccessResponse(msg *nats.Msg, status int, body any) {
	response := &data.Response{
		StatusCode: status,
		Data:       body,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		err := msg.Respond([]byte(`{"status":500,"error":"internal server error"}`))
		if err != nil {
			app.logger.Error("failed to send fallback response", "error", err)
			return
		}
	}

	err = msg.Respond(responseData)
	if err != nil {
		app.logger.Error("failed to send success response", "error", err)
	}
}
