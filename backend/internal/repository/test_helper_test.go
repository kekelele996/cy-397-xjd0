package repository

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{},
		&model.ContractTemplate{},
		&model.Contract{},
		&model.ContractSigner{},
		&model.LegalTicket{},
		&model.TicketReply{},
		&model.KnowledgeFAQ{},
		&model.TemplateFavorite{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
