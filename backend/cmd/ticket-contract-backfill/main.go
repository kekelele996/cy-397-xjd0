// Command ticket-contract-backfill 在数据库升级后执行一次：
// 扫描未登记合同编号的旧工单，按问题描述里的合同编号回填快照；
// 回填不出的工单列在 unresolved 中输出，供人工处理。
//
//	go run ./cmd/ticket-contract-backfill
//
// 也支持指定工单编号：
//
//	go run ./cmd/ticket-contract-backfill -tickets=12,18
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/config"
	"github.com/contractapi/contractapi/internal/repository"
	"github.com/contractapi/contractapi/internal/service"
)

func main() {
	ticketIDsFlag := flag.String("tickets", "", "只回填指定工单编号，逗号分隔；默认扫描全部未关联工单")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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

	ticketRepo := repository.NewTicketRepository(db)
	contractRepo := repository.NewContractRepository(db)
	backfill := service.NewBackfillService(ticketRepo, contractRepo, logger)

	var ticketIDs []uint64
	if strings.TrimSpace(*ticketIDsFlag) != "" {
		for _, raw := range strings.Split(*ticketIDsFlag, ",") {
			id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
			if err != nil || id == 0 {
				logger.Error("invalid ticket id", "value", raw)
				os.Exit(2)
			}
			ticketIDs = append(ticketIDs, id)
		}
	}

	result, err := backfill.Run(ticketIDs)
	if err != nil {
		logger.Error("backfill failed", "error", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "backfilled %d ticket(s):\n", len(result.Backfilled))
	for _, item := range result.Backfilled {
		fmt.Fprintf(os.Stdout, "  ticket %d -> contract %d\n", item.TicketID, item.ContractID)
	}
	fmt.Fprintf(os.Stdout, "unresolved %d ticket(s):\n", len(result.Unresolved))
	for _, item := range result.Unresolved {
		fmt.Fprintf(os.Stdout, "  ticket %d: %s\n", item.TicketID, item.Reason)
	}
}

func openDatabase(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
		if err == nil {
			if sqlDB, pingErr := db.DB(); pingErr == nil {
				if pingErr = sqlDB.Ping(); pingErr == nil {
					return db, nil
				}
			}
		}
		logger.Warn("waiting for database", "attempt", attempt, "error", err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("connect database: %w", err)
}
