package model

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Route é a representação de domínio de uma rota da API
type Route struct {
	Path            string        // O caminho da rota ex: /api/users
	ServiceURL      string        // A URL do serviço de backend
	Methods         []string      // Métodos HTTP permitidos
	Headers         []string      // Cabeçalhos a serem passados
	Description     string        // Descrição da rota
	IsActive        bool          // Se a rota está ativa
	CallCount       int64         // Número de chamadas realizadas
	TotalResponse   time.Duration // Tempo total de resposta
	RequiredHeaders []string      // Cabeçalhos obrigatórios
	CreatedAt       time.Time     // Data de criação
	UpdatedAt       time.Time     // Data de atualização
}

// AverageResponseTime calcula o tempo médio de resposta
func (r *Route) AverageResponseTime() time.Duration {
	if r.CallCount == 0 {
		return 0
	}
	return r.TotalResponse / time.Duration(r.CallCount)
}

// IsMethodAllowed verifica se um método é permitido
func (r *Route) IsMethodAllowed(method string) bool {
	for _, m := range r.Methods {
		if m == method {
			return true
		}
	}
	return false
}

// HasRequiredHeaders verifica se todos os cabeçalhos obrigatórios estão presentes
func (r *Route) HasRequiredHeaders(headers map[string]string) bool {
	for _, required := range r.RequiredHeaders {
		if _, ok := headers[required]; !ok {
			return false
		}
	}
	return true
}

// Validate verifica se a rota é válida
func (r *Route) Validate() error {
	if r.Path == "" {
		return errors.New("path é obrigatório")
	}
	if r.ServiceURL == "" {
		return errors.New("serviceURL é obrigatório")
	}
	if len(r.Methods) == 0 {
		return errors.New("ao menos um método HTTP é obrigatório")
	}

	// Validar URL do serviço
	_, err := url.Parse(r.ServiceURL)
	if err != nil {
		return fmt.Errorf("serviceURL inválida: %w", err)
	}

	return nil
}

func MatchRoutePath(registeredPath, requestPath string) bool {
	// Correspondência exata
	if registeredPath == requestPath {
		return true
	}

	// Verificar correspondência de wildcard
	if strings.HasSuffix(registeredPath, "/*") {
		prefix := strings.TrimSuffix(registeredPath, "/*")
		return strings.HasPrefix(requestPath, prefix)
	}

	// Verificar formato com placeholders (ex: /weather/:cep)
	if strings.Contains(registeredPath, ":") {
		regParts := strings.Split(registeredPath, "/")
		reqParts := strings.Split(requestPath, "/")

		if len(regParts) != len(reqParts) {
			return false
		}

		for i := 0; i < len(regParts); i++ {
			if strings.HasPrefix(regParts[i], ":") {
				// Placeholder, aceita qualquer valor
				continue
			}

			if regParts[i] != reqParts[i] {
				return false
			}
		}

		return true
	}

	return false
}
