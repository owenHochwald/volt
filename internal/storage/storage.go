package storage

import (
	"github.com/owenHochwald/Volt/internal/http"
)

type Storage interface {
	Save(requests *http.Request) error
	Load() ([]http.Request, error)
	Delete(id int64) error
	GetAllURLs() ([]string, error)
}
