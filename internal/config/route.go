package config

import "time"

type Route struct {
	Path          string        `json:"path"`
	ServiceURL    string        `json:"serviceURL"`
	Methods       []string      `json:"methods"`
	Headers       []string      `json:"headers"`
	Description   string        `json:"description"`
	IsActive      bool          `json:"isActive"`
	CallCount     int64         `json:"callCount"`
	TotalResponse time.Duration `json:"totalResponse"`
}
