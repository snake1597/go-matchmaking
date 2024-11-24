package model

import "go-matchmaking/enum"

type ServerError struct {
	Status  enum.ErrorStatus `json:"status"`
	Message string           `json:"message"`
}

func (s *ServerError) Error() string {
	return s.Message
}

func NewServerError(stauts enum.ErrorStatus, err error) *ServerError {
	return &ServerError{
		Status:  stauts,
		Message: err.Error(),
	}
}
