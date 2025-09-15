package db

import (
	"github.com/danglnh07/zola/util"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Queries struct {
	DB *gorm.DB
}

func NewQueries(config *util.Config) (*Queries, error) {
	// Connect to database
	DB, err := gorm.Open(postgres.Open(config.DBConn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	//Return the queries struct
	return &Queries{
		DB: DB,
	}, nil
}

func (queries *Queries) AutoMigration() error {
	return queries.DB.AutoMigrate(&Account{}, &Message{})
}
