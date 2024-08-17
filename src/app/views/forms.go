package views

import (
	"coachwise/src/app/models"

	"github.com/google/uuid"
)

type OrganizationForm struct {
	Name        string     `json:"name" validate:"required,min=3,max=32"`
	Description string     `json:"description" validate:"required,min=3,max=32"`
	LogoID      *uuid.UUID `json:"logo_id"`
}

type SchemaForm struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Public      bool    `json:"public"`
	Attributes  []struct {
		Name        string               `json:"name"`
		Description *string              `json:"description"`
		Type        models.AttributeType `json:"type"`
	} `json:"attributes"`
}
