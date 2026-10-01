package domain

import "github.com/google/uuid"

// GenerateUUID generates a collision-resistant unique identifier.
func GenerateUUID() string {
	return uuid.New().String()
}
