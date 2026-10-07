package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/handler"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
	"github.com/contractapi/contractapi/internal/router"
	"github.com/contractapi/contractapi/internal/service"
	"github.com/contractapi/contractapi/pkg/jwtutil"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setupRouter(t *testing.T) (*gin.Engine, *gorm.DB, *jwtutil.Manager) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{},
		&model.ContractTemplate{},
		&model.Contract{},
		&model.ContractSigner{},
		&model.LegalTicket{},
		&model.TicketReply{},
		&model.TicketContractBackfail{},
		&model.KnowledgeFAQ{},
		&model.TemplateFavorite{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	logger := testLogger()
	userRepo := repository.NewUserRepository(db)
	templateRepo := repository.NewTemplateRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	contractRepo := repository.NewContractRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	knowledgeRepo := repository.NewKnowledgeRepository(db)

	pdf := service.NewPDFService(logger)
	authSvc := service.NewAuthService(userRepo, jwtManager(), logger)
	templateSvc := service.NewTemplateService(templateRepo, favoriteRepo, logger)
	contractSvc := service.NewContractService(contractRepo, ticketRepo, templateRepo, pdf, logger)
	ticketSvc := service.NewTicketService(ticketRepo, contractRepo, logger)
	knowledgeSvc := service.NewKnowledgeService(knowledgeRepo, logger)

	jwt := jwtManager()
	engine := router.New(
		logger, jwt,
		handler.NewAuthHandler(authSvc, logger),
		handler.NewTemplateHandler(templateSvc, logger),
		handler.NewContractHandler(contractSvc, logger),
		handler.NewTicketHandler(ticketSvc, logger),
		handler.NewKnowledgeHandler(knowledgeSvc, logger),
	)
	return engine, db, jwt
}

func jwtManager() *jwtutil.Manager {
	return jwtutil.NewManager("test-secret", time.Hour)
}

func doJSON(t *testing.T, engine *gin.Engine, method, path, token string, body any) (int, apiEnvelope) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	var env apiEnvelope
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode response %q: %v", rec.Body.String(), err)
		}
	}
	return rec.Code, env
}

func mustToken(t *testing.T, jwt *jwtutil.Manager, userID uint64) string {
	t.Helper()
	token, _, err := jwt.Generate(userID, "tester")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func seedContract(t *testing.T, db *gorm.DB, no, status string) *model.Contract {
	t.Helper()
	c := &model.Contract{ContractNo: no, UserID: 1, TemplateID: 1, Title: "合同", Status: status}
	if err := db.Create(c).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	return c
}

func createTicket(t *testing.T, engine *gin.Engine, token, contractNo string) uint64 {
	t.Helper()
	status, env := doJSON(t, engine, http.MethodPost, "/api/v1/tickets", token, map[string]any{
		"type":        constants.TicketTypeContract,
		"title":       "纠纷",
		"description": "对方违约",
		"contract_no": contractNo,
	})
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("create ticket status=%d env=%+v", status, env)
	}
	var ticket struct {
		ID uint64 `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &ticket); err != nil {
		t.Fatalf("decode ticket: %v", err)
	}
	return ticket.ID
}

func TestTicketCloseReconcileFlow(t *testing.T) {
	engine, db, jwt := setupRouter(t)
	token := mustToken(t, jwt, 1)
	seedContract(t, db, "HT-2026-000001", constants.ContractStatusPendingSign)

	ticketID := createTicket(t, engine, token, "HT-2026-000001")

	// 工单详情实时带出合同现状与登记快照。
	status, env := doJSON(t, engine, http.MethodGet, "/api/v1/tickets/"+itoa(ticketID), token, nil)
	if status != http.StatusOK {
		t.Fatalf("get ticket status=%d", status)
	}
	var detail struct {
		RegisteredContractStatus string `json:"registered_contract_status"`
		Contract                 struct {
			No      string `json:"contract_no"`
			Status  string `json:"status"`
			Found   bool   `json:"found"`
			Signers []any  `json:"signers"`
		} `json:"contract"`
	}
	if err := json.Unmarshal(env.Data, &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if !detail.Contract.Found || detail.Contract.Status != constants.ContractStatusPendingSign ||
		detail.RegisteredContractStatus != constants.ContractStatusPendingSign {
		t.Fatalf("detail = %+v", detail)
	}

	// 合同被签署 -> 关联工单自动待复核。
	if err := db.Model(&model.Contract{}).Where("contract_no = ?", "HT-2026-000001").
		Updates(map[string]any{"status": constants.ContractStatusSigned}).Error; err != nil {
		t.Fatalf("sign contract: %v", err)
	}
	if _, err := repository.NewTicketRepository(db).MarkReviewPendingByContractNo("HT-2026-000001"); err != nil {
		t.Fatalf("mark review pending: %v", err)
	}

	// 对不上时关单：409 + 差异明细，工单保持挂起。
	status, env = doJSON(t, engine, http.MethodPatch,
		"/api/v1/tickets/"+itoa(ticketID)+"/status", token, map[string]any{"status": "closed"})
	if status != http.StatusConflict || env.Code != constants.CodeReconcileFailed {
		t.Fatalf("blocked close status=%d env=%+v", status, env)
	}
	var mismatches []struct {
		Reason           string `json:"reason"`
		RegisteredStatus string `json:"registered_status"`
		CurrentStatus    string `json:"current_status"`
	}
	if err := json.Unmarshal(env.Data, &mismatches); err != nil || len(mismatches) != 1 {
		t.Fatalf("mismatches=%s err=%v", env.Data, err)
	}
	if mismatches[0].Reason != "status_mismatch" ||
		mismatches[0].RegisteredStatus != constants.ContractStatusPendingSign ||
		mismatches[0].CurrentStatus != constants.ContractStatusSigned {
		t.Fatalf("mismatch = %+v", mismatches[0])
	}
}

func TestBatchCloseAndBackfill(t *testing.T) {
	engine, db, jwt := setupRouter(t)
	token := mustToken(t, jwt, 1)
	seedContract(t, db, "HT-2026-000007", constants.ContractStatusSigned)
	seedContract(t, db, "HT-2026-000008", constants.ContractStatusSigned)

	matched := createTicket(t, engine, token, "HT-2026-000007")
	stale := createTicket(t, engine, token, "HT-2026-000008")
	if err := db.Model(&model.Contract{}).Where("contract_no = ?", "HT-2026-000008").
		Update("status", constants.ContractStatusExpired).Error; err != nil {
		t.Fatalf("expire contract: %v", err)
	}

	// 旧数据：一笔描述带编号，一笔没有编号。
	legacyOK := &model.LegalTicket{UserID: 1, Type: "contract", Title: "旧",
		Description: "编号 HT-2026-000007 的合同纠纷", Status: "pending"}
	legacyMiss := &model.LegalTicket{UserID: 1, Type: "other", Title: "旧2",
		Description: "没有任何编号", Status: "pending"}
	if err := db.Create(legacyOK).Error; err != nil || db.Create(legacyMiss).Error != nil {
		t.Fatalf("seed legacy tickets")
	}

	status, env := doJSON(t, engine, http.MethodPost, "/api/v1/admin/tickets/batch-close", token,
		map[string]any{"ticket_ids": []uint64{matched, stale}})
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("batch close status=%d env=%+v", status, env)
	}
	var result struct {
		Closed    []uint64 `json:"closed"`
		Suspended []struct {
			TicketID uint64 `json:"ticket_id"`
			Reason   string `json:"reason"`
		} `json:"suspended"`
	}
	if err := json.Unmarshal(env.Data, &result); err != nil {
		t.Fatalf("decode batch result: %v", err)
	}
	if len(result.Closed) != 1 || result.Closed[0] != matched {
		t.Fatalf("closed = %v", result.Closed)
	}
	if len(result.Suspended) != 1 || result.Suspended[0].TicketID != stale ||
		result.Suspended[0].Reason != "status_mismatch" {
		t.Fatalf("suspended = %+v", result.Suspended)
	}

	// 回填：legacyOK 回填成功，legacyMiss 进异常清单。
	status, env = doJSON(t, engine, http.MethodPost, "/api/v1/admin/tickets/backfill-contract", token, nil)
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("backfill status=%d env=%+v", status, env)
	}
	var bf struct {
		Backfilled int `json:"backfilled"`
		Failed     int `json:"failed"`
	}
	if err := json.Unmarshal(env.Data, &bf); err != nil || bf.Backfilled != 1 || bf.Failed != 1 {
		t.Fatalf("backfill result = %s, err=%v", env.Data, err)
	}

	status, env = doJSON(t, engine, http.MethodGet, "/api/v1/admin/tickets/backfails", token, nil)
	if status != http.StatusOK {
		t.Fatalf("list backfails status=%d", status)
	}
	var page struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(env.Data, &page); err != nil || page.Total != 1 {
		t.Fatalf("backfails page = %s, err=%v", env.Data, err)
	}

	// 未带 token 访问管理接口被拒。
	if code, _ := doJSON(t, engine, http.MethodPost, "/api/v1/admin/tickets/batch-close", "",
		map[string]any{"ticket_ids": []uint64{matched}}); code != http.StatusUnauthorized {
		t.Fatalf("batch close without token code=%d, want 401", code)
	}
}
