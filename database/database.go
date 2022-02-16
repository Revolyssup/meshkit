package database

import (
	"sync"

	"github.com/meshery/meshkit/logger"
	sqlite "gorm.io/driver/sqlite"
	gormpkg "gorm.io/gorm"
)

const (
	POSTGRES = "postgres"
	SQLITE   = "sqlite"
)

type Options struct {
	Filename string `json:"filename,omitempty"`
	Engine   string `json:"engine,omitempty"`
	Logger   logger.Handler
}

type Model struct {
	ID        string `json:"id,omitempty" gorm:"primarykey"`
	CreatedAt string `json:"created_at,omitempty" gorm:"index"`
	UpdatedAt string `json:"updated_at,omitempty" gorm:"index"`
	DeletedAt string `json:"deleted_at,omitempty" gorm:"index"`
}

type Handler struct {
	*gormpkg.DB
	*sync.Mutex
	// Implement methods if necessary
}

func (h *Handler) DBClose() error {
	db, err := h.DB.DB()
	if err != nil {
		return err
	}
	err = db.Close() //It ensures that all writes have completed and the database is not corrupted.
	if err != nil {
		return err
	}
	return nil
}
func New(opts Options) (Handler, error) {
	switch opts.Engine {
	case POSTGRES:
		return Handler{}, ErrNoneDatabase
	case SQLITE:
		config := &gormpkg.Config{}
		if opts.Logger != nil {
			config.Logger = opts.Logger.DatabaseLogger()
		}

		db, err := gormpkg.Open(sqlite.Open(opts.Filename), config)
		if err != nil {
			return Handler{}, ErrDatabaseOpen(err)
		}

		return Handler{
			db,
			&sync.Mutex{},
		}, nil
	}

	return Handler{}, ErrNoneDatabase
}
