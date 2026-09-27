package domain

import "github.com/google/uuid"

type User struct {
	ID     uuid.UUID `json:"id" gorm:"primaryKey;default:gen_random_uuid()"`
	Name   string    `json:"name"`
	Graphs []Graph   `json:"graphs,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
