package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Tipos de erro comuns
var (
	ErrNotFound           = errors.New("recurso não encontrado")
	ErrBadRequest         = errors.New("requisição inválida")
	ErrUnauthorized       = errors.New("não autorizado")
	ErrForbidden          = errors.New("acesso negado")
	ErrInternalServer     = errors.New("erro interno do servidor")
	ErrServiceUnavailable = errors.New("serviço indisponível")
	ErrTimeout            = errors.New("tempo de espera excedido")
	ErrDuplicate          = errors.New("recurso já existe")
)

// APIError representa um erro da API com informações adicionais
type APIError struct {
	Code        int         `json:"-"`
	Message     string      `json:"message"`
	Details     interface{} `json:"details,omitempty"`
	OriginalErr error       `json:"-"`
}

// Error implementa a interface error
func (e *APIError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.OriginalErr)
	}
	return e.Message
}

// Unwrap permite usar errors.Is e errors.As
func (e *APIError) Unwrap() error {
	return e.OriginalErr
}

// New cria um novo APIError
func New(code int, message string, err error) *APIError {
	return &APIError{
		Code:        code,
		Message:     message,
		OriginalErr: err,
	}
}

// WithDetails adiciona detalhes ao erro
func (e *APIError) WithDetails(details interface{}) *APIError {
	e.Details = details
	return e
}

// NotFound cria um erro 404
func NotFound(resource string, err error) *APIError {
	message := fmt.Sprintf("%s não encontrado", resource)
	return New(http.StatusNotFound, message, err)
}

// BadRequest cria um erro 400
func BadRequest(message string, err error) *APIError {
	return New(http.StatusBadRequest, message, err)
}

// Unauthorized cria um erro 401
func Unauthorized(message string, err error) *APIError {
	if message == "" {
		message = "Autenticação necessária"
	}
	return New(http.StatusUnauthorized, message, err)
}

// Forbidden cria um erro 403
func Forbidden(message string, err error) *APIError {
	if message == "" {
		message = "Acesso negado"
	}
	return New(http.StatusForbidden, message, err)
}

// InternalServer cria um erro 500
func InternalServer(message string, err error) *APIError {
	if message == "" {
		message = "Erro interno do servidor"
	}
	return New(http.StatusInternalServerError, message, err)
}
