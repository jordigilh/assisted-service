// Code generated by go-swagger; DO NOT EDIT.

package installer

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/openshift/assisted-service/models"
)

// ResetHostValidationOKCode is the HTTP code returned for type ResetHostValidationOK
const ResetHostValidationOKCode int = 200

/*ResetHostValidationOK Success.

swagger:response resetHostValidationOK
*/
type ResetHostValidationOK struct {

	/*
	  In: Body
	*/
	Payload *models.Host `json:"body,omitempty"`
}

// NewResetHostValidationOK creates ResetHostValidationOK with default headers values
func NewResetHostValidationOK() *ResetHostValidationOK {

	return &ResetHostValidationOK{}
}

// WithPayload adds the payload to the reset host validation o k response
func (o *ResetHostValidationOK) WithPayload(payload *models.Host) *ResetHostValidationOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the reset host validation o k response
func (o *ResetHostValidationOK) SetPayload(payload *models.Host) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ResetHostValidationOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// ResetHostValidationBadRequestCode is the HTTP code returned for type ResetHostValidationBadRequest
const ResetHostValidationBadRequestCode int = 400

/*ResetHostValidationBadRequest Bad Request

swagger:response resetHostValidationBadRequest
*/
type ResetHostValidationBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewResetHostValidationBadRequest creates ResetHostValidationBadRequest with default headers values
func NewResetHostValidationBadRequest() *ResetHostValidationBadRequest {

	return &ResetHostValidationBadRequest{}
}

// WithPayload adds the payload to the reset host validation bad request response
func (o *ResetHostValidationBadRequest) WithPayload(payload *models.Error) *ResetHostValidationBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the reset host validation bad request response
func (o *ResetHostValidationBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ResetHostValidationBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// ResetHostValidationUnauthorizedCode is the HTTP code returned for type ResetHostValidationUnauthorized
const ResetHostValidationUnauthorizedCode int = 401

/*ResetHostValidationUnauthorized Unauthorized.

swagger:response resetHostValidationUnauthorized
*/
type ResetHostValidationUnauthorized struct {

	/*
	  In: Body
	*/
	Payload *models.InfraError `json:"body,omitempty"`
}

// NewResetHostValidationUnauthorized creates ResetHostValidationUnauthorized with default headers values
func NewResetHostValidationUnauthorized() *ResetHostValidationUnauthorized {

	return &ResetHostValidationUnauthorized{}
}

// WithPayload adds the payload to the reset host validation unauthorized response
func (o *ResetHostValidationUnauthorized) WithPayload(payload *models.InfraError) *ResetHostValidationUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the reset host validation unauthorized response
func (o *ResetHostValidationUnauthorized) SetPayload(payload *models.InfraError) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ResetHostValidationUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// ResetHostValidationForbiddenCode is the HTTP code returned for type ResetHostValidationForbidden
const ResetHostValidationForbiddenCode int = 403

/*ResetHostValidationForbidden Forbidden.

swagger:response resetHostValidationForbidden
*/
type ResetHostValidationForbidden struct {

	/*
	  In: Body
	*/
	Payload *models.InfraError `json:"body,omitempty"`
}

// NewResetHostValidationForbidden creates ResetHostValidationForbidden with default headers values
func NewResetHostValidationForbidden() *ResetHostValidationForbidden {

	return &ResetHostValidationForbidden{}
}

// WithPayload adds the payload to the reset host validation forbidden response
func (o *ResetHostValidationForbidden) WithPayload(payload *models.InfraError) *ResetHostValidationForbidden {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the reset host validation forbidden response
func (o *ResetHostValidationForbidden) SetPayload(payload *models.InfraError) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ResetHostValidationForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(403)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// ResetHostValidationNotFoundCode is the HTTP code returned for type ResetHostValidationNotFound
const ResetHostValidationNotFoundCode int = 404

/*ResetHostValidationNotFound Error.

swagger:response resetHostValidationNotFound
*/
type ResetHostValidationNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewResetHostValidationNotFound creates ResetHostValidationNotFound with default headers values
func NewResetHostValidationNotFound() *ResetHostValidationNotFound {

	return &ResetHostValidationNotFound{}
}

// WithPayload adds the payload to the reset host validation not found response
func (o *ResetHostValidationNotFound) WithPayload(payload *models.Error) *ResetHostValidationNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the reset host validation not found response
func (o *ResetHostValidationNotFound) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ResetHostValidationNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// ResetHostValidationConflictCode is the HTTP code returned for type ResetHostValidationConflict
const ResetHostValidationConflictCode int = 409

/*ResetHostValidationConflict Error.

swagger:response resetHostValidationConflict
*/
type ResetHostValidationConflict struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewResetHostValidationConflict creates ResetHostValidationConflict with default headers values
func NewResetHostValidationConflict() *ResetHostValidationConflict {

	return &ResetHostValidationConflict{}
}

// WithPayload adds the payload to the reset host validation conflict response
func (o *ResetHostValidationConflict) WithPayload(payload *models.Error) *ResetHostValidationConflict {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the reset host validation conflict response
func (o *ResetHostValidationConflict) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ResetHostValidationConflict) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(409)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// ResetHostValidationInternalServerErrorCode is the HTTP code returned for type ResetHostValidationInternalServerError
const ResetHostValidationInternalServerErrorCode int = 500

/*ResetHostValidationInternalServerError Error.

swagger:response resetHostValidationInternalServerError
*/
type ResetHostValidationInternalServerError struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewResetHostValidationInternalServerError creates ResetHostValidationInternalServerError with default headers values
func NewResetHostValidationInternalServerError() *ResetHostValidationInternalServerError {

	return &ResetHostValidationInternalServerError{}
}

// WithPayload adds the payload to the reset host validation internal server error response
func (o *ResetHostValidationInternalServerError) WithPayload(payload *models.Error) *ResetHostValidationInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the reset host validation internal server error response
func (o *ResetHostValidationInternalServerError) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ResetHostValidationInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
