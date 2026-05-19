// Mock Vendotek EzPOS terminal.
//
// Реализует HTTP/JSON сервер по протоколу EzPOS v1.5.
// Поддерживаемые эндпоинты:
//
//	POST /async/cashless/sale[/card|/qr] — создать операцию
//	GET  /sale?id=...                    — опросить статус
//	POST /async/cashless/sale/cancel?id= — запрос отмены
//	POST /async/cashless/reversal?id=    — запрос возврата
//	POST /async/fiscal?id=               — заглушка 405 (вариант B: фискализация на нашей ККТ)
//	GET  /status                         — состояние терминала
//	POST /show/qr, POST /screen          — заглушки 200
//
// Debug-эндпоинты (не часть EzPOS, только для разработчика):
//
//	POST /debug/ops/:id/approve|decline|reverted|cancel
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type opStatus string

const (
	opCreated    opStatus = "created"
	opWaitCard   opStatus = "wait_for_card"
	opInProgress opStatus = "in_progress"
	opCompleted  opStatus = "completed"
	opReverted   opStatus = "reverted"
	opFail       opStatus = "fail"
)

type scenario string

const (
	scenarioSuccess  scenario = "success"
	scenarioDecline  scenario = "decline"
	scenarioTimeout  scenario = "timeout"
	scenarioReverted scenario = "reverted"
)

type slip struct {
	PAN          string `json:"pan"`
	RRN          string `json:"rrn"`
	ApprovalCode string `json:"approval_code"`
	Amount       int64  `json:"amount"`
	Date         string `json:"date"`
	POSEntryMode string `json:"pos_entry_mode"`
	AppLabel     string `json:"app_label"`
}

type saleReq struct {
	ID       string `json:"id"`
	Sum      int64  `json:"sum"`
	Currency string `json:"currency"`
}

type saleResp struct {
	ID     string   `json:"id"`
	Status opStatus `json:"status"`
	Info   string   `json:"info,omitempty"`
	Slip   *slip    `json:"slip,omitempty"`
}

type storedOp struct {
	id          string
	sum         int64
	currency    string
	status      opStatus
	info        string
	sl          *slip
	waitCardAt  time.Time
	progressAt  time.Time
	finalizeAt  time.Time
	autoOutcome opStatus
}

type mockConfig struct {
	defaultScenario  scenario
	autoWaitMS       int
	autoDelayMS      int
	timeoutMS        int
	randomDeclinePct int
	serialNumber     string
	debug            bool
}

type store struct {
	mu       sync.Mutex
	ops      map[string]*storedOp
	lastOpID string
	cfg      mockConfig
	rng      *rand.Rand
}

func newStore(cfg mockConfig) *store {
	return &store{
		ops: make(map[string]*storedOp),
		cfg: cfg,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *store) create(id string, sum int64, currency string) (*saleResp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.ops[id]; exists {
		return nil, &apiErr{http.StatusConflict, "operation already exists"}
	}

	now := time.Now()
	waitDur := time.Duration(s.cfg.autoWaitMS) * time.Millisecond
	outcome := s.chooseOutcome()

	var delayDur time.Duration
	if outcome == opFail && s.cfg.defaultScenario == scenarioTimeout {
		delayDur = time.Duration(s.cfg.timeoutMS) * time.Millisecond
	} else {
		delayDur = time.Duration(s.cfg.autoDelayMS) * time.Millisecond
	}

	op := &storedOp{
		id:          id,
		sum:         sum,
		currency:    normalizeCurrency(currency),
		status:      opCreated,
		waitCardAt:  now.Add(waitDur),
		progressAt:  now.Add(waitDur + delayDur/2),
		finalizeAt:  now.Add(waitDur + delayDur),
		autoOutcome: outcome,
	}
	s.ops[id] = op
	s.lastOpID = id

	if s.cfg.debug {
		log.Printf("vendotek mock: CREATE id=%s sum=%d outcome=%s", id, sum, outcome)
	}
	return opToResp(op), nil
}

func (s *store) get(id string) (*saleResp, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.ops[id]
	if !ok {
		return nil, false
	}
	s.tick(op)
	return opToResp(op), true
}

func (s *store) cancel(id string) (*saleResp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.ops[id]
	if !ok {
		return nil, &apiErr{http.StatusNotFound, "operation not found"}
	}
	s.tick(op)

	if !isFinal(op.status) {
		op.status = opFail
		op.info = "cancelled by control device"
		op.autoOutcome = ""
		if s.cfg.debug {
			log.Printf("vendotek mock: CANCEL id=%s", id)
		}
	}
	return opToResp(op), nil
}

func (s *store) reversal(id string) (*saleResp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.ops[id]
	if !ok {
		return nil, &apiErr{http.StatusNotFound, "operation not found"}
	}
	s.tick(op)

	if op.status == opCompleted {
		op.status = opReverted
		op.info = "reversal accepted"
		if s.cfg.debug {
			log.Printf("vendotek mock: REVERSAL id=%s", id)
		}
	}
	return opToResp(op), nil
}

func (s *store) manualFinalize(id string, status opStatus, info string) (*saleResp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.ops[id]
	if !ok {
		return nil, &apiErr{http.StatusNotFound, "operation not found"}
	}
	if isFinal(op.status) {
		return opToResp(op), nil
	}

	op.status = status
	op.info = info
	op.autoOutcome = ""
	if status == opCompleted || status == opReverted {
		op.sl = s.genSlip(op)
	}
	if s.cfg.debug {
		log.Printf("vendotek mock: MANUAL id=%s status=%s", id, status)
	}
	return opToResp(op), nil
}

func (s *store) statusInfo() gin.H {
	s.mu.Lock()
	defer s.mu.Unlock()

	termStatus := "ok"
	for _, op := range s.ops {
		s.tick(op)
		if !isFinal(op.status) {
			termStatus = "busy"
			break
		}
	}
	return gin.H{
		"status":     termStatus,
		"S/N":        s.cfg.serialNumber,
		"info":       fmt.Sprintf("EzPOS mock, scenario=%s", s.cfg.defaultScenario),
		"last_op_id": s.lastOpID,
	}
}

func (s *store) tick(op *storedOp) {
	if isFinal(op.status) {
		return
	}
	now := time.Now()
	if op.status == opCreated && !now.Before(op.waitCardAt) {
		op.status = opWaitCard
	}
	if op.status == opWaitCard && !now.Before(op.progressAt) {
		op.status = opInProgress
	}
	if op.status == opInProgress && !now.Before(op.finalizeAt) {
		s.finalize(op)
	}
}

func (s *store) finalize(op *storedOp) {
	op.status = op.autoOutcome
	switch op.autoOutcome {
	case opCompleted:
		op.info = "approved"
		op.sl = s.genSlip(op)
	case opFail:
		if s.cfg.defaultScenario == scenarioTimeout {
			op.info = "operation timeout"
		} else {
			op.info = "declined by mock scenario"
		}
	case opReverted:
		op.info = "reverted"
		op.sl = s.genSlip(op)
	}
	op.autoOutcome = ""
}

func (s *store) genSlip(op *storedOp) *slip {
	return &slip{
		PAN:          "411111******1111",
		RRN:          fmt.Sprintf("%012d", time.Now().UnixMilli()%1_000_000_000_000),
		ApprovalCode: fmt.Sprintf("%06d", s.rng.Intn(1_000_000)),
		Amount:       op.sum,
		Date:         time.Now().Format(time.RFC3339),
		POSEntryMode: "07",
		AppLabel:     "VISA",
	}
}

func (s *store) chooseOutcome() opStatus {
	switch s.cfg.defaultScenario {
	case scenarioSuccess:
		return opCompleted
	case scenarioDecline:
		return opFail
	case scenarioReverted:
		return opReverted
	case scenarioTimeout:
		return opFail
	default:
		if s.rng.Intn(100) < s.cfg.randomDeclinePct {
			return opFail
		}
		return opCompleted
	}
}

func opToResp(op *storedOp) *saleResp {
	return &saleResp{
		ID:     op.id,
		Status: op.status,
		Info:   op.info,
		Slip:   op.sl,
	}
}

func isFinal(s opStatus) bool {
	return s == opCompleted || s == opReverted || s == opFail
}

func normalizeCurrency(s string) string {
	v := strings.ToUpper(strings.TrimSpace(s))
	if v == "" {
		return "RUB"
	}
	return v
}

type apiErr struct {
	code int
	msg  string
}

func (e *apiErr) Error() string { return e.msg }

func main() {
	loadEnv()
	cfg := readConfig()
	s := newStore(cfg)

	gin.SetMode(envStr("GIN_MODE", gin.DebugMode))

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	saleHandler := func(c *gin.Context) {
		var req saleReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		req.ID = strings.TrimSpace(req.ID)
		if req.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}
		if cfg.debug {
			raw, _ := json.Marshal(req)
			log.Printf("vendotek mock: POST /sale body=%s", raw)
		}
		resp, err := s.create(req.ID, req.Sum, req.Currency)
		if err != nil {
			renderErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}

	r.POST("/async/cashless/sale", saleHandler)
	r.POST("/async/cashless/sale/card", saleHandler)
	r.POST("/async/cashless/sale/qr", saleHandler)

	r.GET("/sale", func(c *gin.Context) {
		id := strings.TrimSpace(c.Query("id"))
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}
		resp, ok := s.get(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "operation not found"})
			return
		}
		if cfg.debug {
			raw, _ := json.Marshal(resp)
			log.Printf("vendotek mock: GET /sale id=%s resp=%s", id, raw)
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/async/cashless/sale/cancel", func(c *gin.Context) {
		id := strings.TrimSpace(c.Query("id"))
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}
		resp, err := s.cancel(id)
		if err != nil {
			renderErr(c, err)
			return
		}
		if cfg.debug {
			log.Printf("vendotek mock: POST /sale/cancel id=%s result=%s", id, resp.Status)
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/async/cashless/reversal", func(c *gin.Context) {
		id := strings.TrimSpace(c.Query("id"))
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}
		resp, err := s.reversal(id)
		if err != nil {
			renderErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/async/fiscal", func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "fiscal not configured on terminal (variant B)"})
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, s.statusInfo())
	})

	r.POST("/show/qr", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/screen", func(c *gin.Context) { c.Status(http.StatusOK) })

	r.POST("/debug/ops/:id/approve", func(c *gin.Context) {
		resp, err := s.manualFinalize(c.Param("id"), opCompleted, "approved manually")
		if err != nil {
			renderErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})
	r.POST("/debug/ops/:id/decline", func(c *gin.Context) {
		resp, err := s.manualFinalize(c.Param("id"), opFail, "declined manually")
		if err != nil {
			renderErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})
	r.POST("/debug/ops/:id/reverted", func(c *gin.Context) {
		resp, err := s.manualFinalize(c.Param("id"), opReverted, "reverted manually")
		if err != nil {
			renderErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})
	r.POST("/debug/ops/:id/cancel", func(c *gin.Context) {
		resp, err := s.manualFinalize(c.Param("id"), opFail, "cancelled manually")
		if err != nil {
			renderErr(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "mock-vendotek-ezpos",
			"time":    time.Now().Format(time.RFC3339),
			"config": gin.H{
				"defaultScenario":  cfg.defaultScenario,
				"autoWaitMS":       cfg.autoWaitMS,
				"autoDelayMS":      cfg.autoDelayMS,
				"randomDeclinePct": cfg.randomDeclinePct,
				"serialNumber":     cfg.serialNumber,
			},
		})
	})

	port := envStr("PORT", "8082")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("mock vendotek (EzPOS): started on :%s", port)
	log.Printf("mock vendotek: scenario=%s wait=%dms delay=%dms decline_pct=%d%% S/N=%s debug=%v",
		cfg.defaultScenario, cfg.autoWaitMS, cfg.autoDelayMS,
		cfg.randomDeclinePct, cfg.serialNumber, cfg.debug)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("mock vendotek: failed: %v", err)
	}
}

func loadEnv() {
	candidates := []string{
		"server/mocks/vendotek/.env",
		"mocks/vendotek/.env",
		".env",
	}
	for _, p := range candidates {
		if err := godotenv.Load(p); err == nil {
			log.Printf("mock vendotek: loaded env from %s", p)
			return
		}
	}
	log.Printf("mock vendotek: .env not found, using system environment")
}

func readConfig() mockConfig {
	return mockConfig{
		defaultScenario:  parseScenario(envStr("VENDOTEK_DEFAULT_SCENARIO", string(scenarioSuccess))),
		autoWaitMS:       envInt("VENDOTEK_AUTO_WAIT_MS", 500),
		autoDelayMS:      envInt("VENDOTEK_AUTO_DELAY_MS", 1500),
		timeoutMS:        envInt("VENDOTEK_TIMEOUT_MS", 600000),
		randomDeclinePct: clamp(envInt("VENDOTEK_RANDOM_DECLINE_PCT", 0), 0, 100),
		serialNumber:     envStr("VENDOTEK_SERIAL_NUMBER", "MOCKVTK0001"),
		debug:            envBool("MOCK_VENDOTEK_DEBUG", false),
	}
}

func parseScenario(raw string) scenario {
	s := scenario(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case scenarioSuccess, scenarioDecline, scenarioTimeout, scenarioReverted:
		return s
	}
	log.Printf("mock vendotek: unknown scenario %q, using %q", raw, scenarioSuccess)
	return scenarioSuccess
}

func renderErr(c *gin.Context, err error) {
	if e, ok := err.(*apiErr); ok {
		c.JSON(e.code, gin.H{"error": e.msg})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}

func envStr(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}

func envInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("mock vendotek: %s=%q is not int, using default %d", name, v, def)
		return def
	}
	return i
}

func envBool(name string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch v {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	log.Printf("mock vendotek: %s=%q is not bool, using default %v", name, v, def)
	return def
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
