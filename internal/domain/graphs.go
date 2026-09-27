package domain

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Graph struct {
	ID         uuid.UUID `json:"id" gorm:"primaryKey;default:gen_random_uuid()"`
	Name       string    `json:"name"`
	UserID     uuid.UUID `json:"user_id"`
	IsDirected bool      `json:"is_directed"`

	Nodes []Node `json:"nodes,omitempty" gorm:"foreignKey:GraphID;constraint:OnDelete:CASCADE"`
	Edges []Edge `json:"edges,omitempty" gorm:"foreignKey:GraphID;constraint:OnDelete:CASCADE"`
}

type Node struct {
	ID       uuid.UUID      `json:"id" gorm:"primaryKey;default:gen_random_uuid()"`
	GraphID  uuid.UUID      `json:"graph_id"`
	Metadata datatypes.JSON `json:"metadata"`
	X        float32        `json:"x"`
	Y        float32        `json:"y"`
}

type Edge struct {
	ID       uuid.UUID      `json:"id" gorm:"primaryKey;default:gen_random_uuid()"`
	GraphID  uuid.UUID      `json:"graph_id"`
	SourceID uuid.UUID      `json:"source_id"`
	TargetID uuid.UUID      `json:"target_id"`
	Metadata datatypes.JSON `json:"metadata"`
	Weight   float64        `json:"weight"`

	Source Node `json:"-" gorm:"foreignKey:SourceID;references:ID;constraint:OnDelete:CASCADE"`
	Target Node `json:"-" gorm:"foreignKey:TargetID;references:ID;constraint:OnDelete:CASCADE"`
}
