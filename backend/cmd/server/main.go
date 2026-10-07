package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/config"
	"github.com/contractapi/contractapi/internal/handler"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
	"github.com/contractapi/contractapi/internal/router"
	"github.com/contractapi/contractapi/internal/service"
	"github.com/contractapi/contractapi/pkg/jwtutil"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	db, err := openDatabase(cfg, logger)
	if err != nil {
		logger.Error("open database failed", "error", err)
		os.Exit(1)
	}

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
		logger.Error("auto migrate failed", "error", err)
		os.Exit(1)
	}

	jwtManager := jwtutil.NewManager(cfg.JWTSecret, time.Duration(cfg.JWTExpireHours)*time.Hour)

	userRepo := repository.NewUserRepository(db)
	templateRepo := repository.NewTemplateRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	contractRepo := repository.NewContractRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	knowledgeRepo := repository.NewKnowledgeRepository(db)

	pdfService := service.NewPDFService(logger)
	authService := service.NewAuthService(userRepo, jwtManager, logger)
	templateService := service.NewTemplateService(templateRepo, favoriteRepo, logger)

	// 工单-合同联动服务：建单登记快照、合同落定联动、关单对账、旧数据回填。
	// 先建联动服务（TicketSyncer），再注入合同服务；合同服务同时作为合同快照读取方。
	ticketContractService := service.NewTicketContractService(ticketRepo, contractRepo, logger)
	contractService := service.NewContractService(contractRepo, templateRepo, pdfService, ticketContractService, logger)
	ticketService := service.NewTicketService(ticketRepo, contractRepo, logger)
	backfillService := service.NewBackfillService(ticketRepo, contractRepo, logger)
	knowledgeService := service.NewKnowledgeService(knowledgeRepo, logger)

	authHandler := handler.NewAuthHandler(authService, logger)
	templateHandler := handler.NewTemplateHandler(templateService, logger)
	contractHandler := handler.NewContractHandler(contractService, logger)
	ticketHandler := handler.NewTicketHandler(ticketService, ticketContractService, backfillService, logger)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeService, logger)

	engine := router.New(logger, jwtManager, authHandler, templateHandler, contractHandler, ticketHandler, knowledgeHandler)

	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.Info("server starting", "addr", addr, "env", cfg.AppEnv)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func openDatabase(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
		if err == nil {
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				if pingErr = sqlDB.Ping(); pingErr == nil {
					logger.Info("database connected", "host", cfg.DBHost, "port", cfg.DBPort, "database", cfg.DBName)
					return db, nil
				}
			}
		}
		logger.Warn("waiting for database", "attempt", attempt, "error", err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("connect database: %w", err)
}
