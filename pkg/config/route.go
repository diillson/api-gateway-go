package config

import (
	"errors"
	"time"
)

type Route struct {
	Path            string        `json:"path" gorm:"type:varchar(255)"`
	ServiceURL      string        `json:"serviceURL" gorm:"type:varchar(255)"`
	Methods         []string      `json:"methods" gorm:"type:json"`
	Headers         []string      `json:"headers" gorm:"type:json"`
	Description     string        `json:"description" gorm:"type:text"`
	IsActive        bool          `json:"isActive"`
	CallCount       int64         `json:"callCount"`
	TotalResponse   time.Duration `json:"totalResponse"`
	RequiredHeaders []string      `json:"requiredHeaders" gorm:"type:json"`
}

func (r *Route) Validate() error {
	if r.Path == "" {
		return errors.New("path is required")
	}
	if r.ServiceURL == "" {
		return errors.New("serviceURL is required")
	}
	if len(r.Methods) == 0 {
		return errors.New("at least one HTTP method is required")
	}
	return nil
}

func (r *Route) IsMethodAllowed(method string) bool {
	for _, m := range r.Methods {
		if m == method {
			return true
		}
	}
	return false
}
