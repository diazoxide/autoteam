package worker

import "github.com/google/uuid"

// UUID helper functions
func UUID() uuid.UUID {
	return uuid.New()
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
