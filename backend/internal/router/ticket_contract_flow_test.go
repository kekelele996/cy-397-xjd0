package router_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/contractapi/contractapi/internal/constants"
	"github.com/contractapi/contractapi/internal/handler"
	"github.com/contractapi/contractapi/internal/middleware"
	"github.com/contractapi/contractapi/internal/model"
	"github.com/contractapi/contractapi/internal/repository"
	"github.com/contractapi/contractapi/internal/service"
	"github.com/contractapi/contractapi/pkg/jwtutil"
)

// newTestRouter 用 SQLite 装配真实仓储/服务与 JWT 中间件，覆盖关单对账全链路。
func newTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{}, &model.ContractTemplate{}, &model.Contract{}, &model.ContractSigner{},
		&model.LegalTicket{}, &model.TicketReply{}, &model.KnowledgeFAQ{}, &model.TemplateFavorite{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	contractRepo := repository.NewContractRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	templateRepo := repository.NewTemplateRepository(db)

	pdf := service.NewPDFService(logger)
	link := service.NewTicketContractService(ticketRepo, contractRepo, logger)
	contractSvc := service.NewContractService(contractRepo, templateRepo, pdf, link, logger)
	ticketSvc := service.NewTicketService(ticketRepo, contractRepo, logger)
	backfillSvc := service.NewBackfillService(ticketRepo, contractRepo, logger)

	ticketH := handler.NewTicketHandler(ticketSvc, link, backfillSvc, logger)
	contractH := handler.NewContractHandler(contractSvc, logger)

	jwtm := jwtutil.NewManager("test-secret", time.Hour)
	token, _, err := jwtm.Generate(1, "tester")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler(logger))
	authed := r.Group("/api/v1").Use(middleware.JWTAuth(jwtm))
	authed.POST("/tickets", ticketH.Create)
	authed.GET("/tickets/:id", ticketH.Get)
	authed.POST("/tickets/:id/close", ticketH.Close)
	authed.POST("/tickets/batch-close", ticketH.BatchClose)
	authed.POST("/tickets/:id/review", ticketH.Review)
	authed.POST("/contracts/:id/sign", contractH.Sign)
	return r, db, token
}

func seedContractAndTicket(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Create(&model.Contract{
		ID: 1, UserID: 1, TemplateID: 1, Title: "合同",
		Status: constants.ContractStatusPendingSign,
	}).Error; err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	if err := db.Create(&model.ContractSigner{ContractID: 1, Name: "张三", Role: "甲方"}).Error; err != nil {
		t.Fatalf("seed signer: %v", err)
	}
	if err := db.Create(&model.LegalTicket{
		UserID:                  1,
		Type:                    constants.TicketTypeContract,
		Title:                   "纠纷",
		Description:             "内容",
		Status:                  constants.TicketStatusReplied,
		ContractID:              1,
		ContractStatusSnapshot:  constants.ContractStatusPendingSign,
		ContractSignersSnapshot: model.SignerSnapshots{{Name: "张三", Role: "甲方"}},
	}).Error; err != nil {
		t.Fatalf("seed ticket: %v", err)
	}
}

func doJSON(t *testing.T, r *gin.Engine, token, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(buf)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCloseReconcileFlow(t *testing.T) {
	r, db, token := newTestRouter(t)
	seedContractAndTicket(t, db)

	// 1) 详情带出合同库现状且 initially 一致。
	w := doJSON(t, r, token, http.MethodGet, "/api/v1/tickets/1", nil)
	var detail struct {
		Data struct {
			Contract struct {
				CurrentStatus  string `json:"current_status"`
				Reconciled     bool   `json:"reconciled"`
				CurrentSigners []struct {
					Name string `json:"name"`
					Role string `json:"role"`
				} `json:"current_signers"`
			} `json:"contract"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal detail: %v body=%s", err, w.Body.String())
	}
	if !detail.Data.Contract.Reconciled ||
		detail.Data.Contract.CurrentStatus != constants.ContractStatusPendingSign ||
		len(detail.Data.Contract.CurrentSigners) != 1 {
		t.Fatalf("detail contract = %+v", detail.Data.Contract)
	}

	// 2) 合同签署落定后，未关工单自动变待复核。
	if w := doJSON(t, r, token, http.MethodPost, "/api/v1/contracts/1/sign",
		map[string]string{"signer_name": "张三", "signer_role": "甲方"}); w.Code != 200 {
		t.Fatalf("sign status = %d body=%s", w.Code, w.Body.String())
	}
	var tk model.LegalTicket
	if err := db.First(&tk, 1).Error; err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if tk.Status != constants.TicketStatusPendingReview {
		t.Fatalf("ticket after sign = %q, want pending_review", tk.Status)
	}

	// 3) 未复核直接关单 -> 409、挂起、差异列出 contract_status。
	w = doJSON(t, r, token, http.MethodPost, "/api/v1/tickets/1/close", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("close status = %d body=%s", w.Code, w.Body.String())
	}
	var bad struct {
		Code int `json:"code"`
		Data struct {
			Mismatches []struct {
				TicketID    uint64 `json:"ticket_id"`
				Differences []struct {
					Field      string `json:"field"`
					Registered any    `json:"registered"`
					Current    any    `json:"current"`
				} `json:"differences"`
			} `json:"mismatches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &bad); err != nil {
		t.Fatalf("unmarshal mismatch: %v", err)
	}
	if bad.Code != constants.CodeReconcileMismatch || len(bad.Data.Mismatches) != 1 ||
		len(bad.Data.Mismatches[0].Differences) < 1 {
		t.Fatalf("mismatch payload = %s", w.Body.String())
	}
	hasStatusDiff := false
	for _, d := range bad.Data.Mismatches[0].Differences {
		if d.Field == "contract_status" {
			hasStatusDiff = true
		}
	}
	if !hasStatusDiff {
		t.Fatalf("expected contract_status difference, got %s", w.Body.String())
	}
	if err := db.First(&tk, 1).Error; err != nil || tk.Status != constants.TicketStatusOnHold {
		t.Fatalf("ticket after failed close = %q, want on_hold", tk.Status)
	}

	// 4) 复核刷新快照后关单成功。
	if w := doJSON(t, r, token, http.MethodPost, "/api/v1/tickets/1/review",
		map[string]string{"note": "已核实"}); w.Code != 200 {
		t.Fatalf("review status = %d body=%s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, token, http.MethodPost, "/api/v1/tickets/1/close", nil)
	if w.Code != 200 {
		t.Fatalf("second close status = %d body=%s", w.Code, w.Body.String())
	}
	if err := db.First(&tk, 1).Error; err != nil || tk.Status != constants.TicketStatusClosed {
		t.Fatalf("ticket after review+close = %q, want closed", tk.Status)
	}
}

func TestBatchCloseListsMismatches(t *testing.T) {
	r, db, token := newTestRouter(t)
	seedContractAndTicket(t, db)

	// 合同置为已过期但工单快照仍是待签署。
	if err := db.Model(&model.Contract{}).Where("id = ?", 1).
		Update("status", constants.ContractStatusExpired).Error; err != nil {
		t.Fatalf("expire contract: %v", err)
	}
	w := doJSON(t, r, token, http.MethodPost, "/api/v1/tickets/batch-close",
		map[string][]uint64{"ticket_ids": {1, 999}})
	if w.Code != 200 {
		t.Fatalf("batch close status = %d body=%s", w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			Closed   []uint64 `json:"closed"`
			Mismatch []struct {
				TicketID uint64 `json:"ticket_id"`
				Reason   string `json:"reason"`
			} `json:"mismatch"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal batch: %v", err)
	}
	if len(res.Data.Closed) != 0 || len(res.Data.Mismatch) != 2 {
		t.Fatalf("batch result = %s", w.Body.String())
	}
	var tk model.LegalTicket
	if err := db.First(&tk, 1).Error; err != nil || tk.Status != constants.TicketStatusOnHold {
		t.Fatalf("ticket = %q, want on_hold", tk.Status)
	}
}
