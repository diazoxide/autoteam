package worker

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSON helper functions for custom types
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(value interface{}, dest interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		if str, ok := value.(string); ok {
			bytes = []byte(str)
		} else {
			return fmt.Errorf("cannot unmarshal non-string/non-bytes value: %T", value)
		}
	}
	return json.Unmarshal(bytes, dest)
}

// GORM hooks for UUID generation
func (w *Worker) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

func (ws *WorkerSettings) BeforeCreate(tx *gorm.DB) error {
	if ws.ID == uuid.Nil {
		ws.ID = uuid.New()
	}
	return nil
}

func (fs *FlowStep) BeforeCreate(tx *gorm.DB) error {
	if fs.ID == uuid.Nil {
		fs.ID = uuid.New()
	}
	return nil
}
