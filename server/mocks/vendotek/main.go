package main

import (
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

type SessionStatus string

const (
	// Статусы жизненного цикла платежной сессии
	SessionStatusCreated    SessionStatus = "created"
	SessionStatusPending    SessionStatus = "pending"
	SessionStatusProcessing SessionStatus = "processing"
	SessionStatusApproved   SessionStatus = "approved"
	SessionStatusDeclined   SessionStatus = "declined"
	SessionStatusCancelled  SessionStatus = "cancelled"
	SessionStatusTimeout    SessionStatus = "timeout"
)

type Scenario string

const (
	// Сценарии для эмуляции разных ответов терминала
	ScenarioApproved Scenario = "approved"
	ScenarioDeclined Scenario = "declined"
	ScenarioTimeout  Scenario = "timeout"
	ScenarioRandom   Scenario = "random"
)

type Session struct {
	// Данные, которые возвращаются по API
	ID                    string        `json:"sessionId"`
	ExternalTransactionID string        `json:"externalTransactionId,omitempty"`
	AmountMinor           int64         `json:"amountMinor"`
	Currency              string        `json:"currency"`
	Status                SessionStatus `json:"status"`
	Scenario              Scenario      `json:"scenario"`
	Error                 string        `json:"error,omitempty"`
	CreatedAt             time.Time     `json:"createdAt"`
	UpdatedAt             time.Time     `json:"updatedAt"`
	StartedAt             *time.Time    `json:"startedAt,omitempty"`
}

type storedSession struct {
	// Внутреннее представление сессии в памяти
	Session
	autoOutcome  SessionStatus
	autoComplete time.Time
}

type eventRecord struct {
	// История действий для отладки
	SessionID string    `json:"sessionId,omitempty"`
	Type      string    `json:"type"`
	Details   string    `json:"details,omitempty"`
	Time      time.Time `json:"time"`
}

type mockConfig struct {
	// Настройки поведения мок сервиса
	DefaultScenario  Scenario `json:"defaultScenario"`
	AutoDelayMS      int      `json:"autoDelayMs"`
	RandomDeclinePct int      `json:"randomDeclinePct"`
}

type store struct {
	// Потокобезопасное in-memory хранилище
	mu       sync.RWMutex
	sessions map[string]*storedSession
	events   []eventRecord
	counter  int64
	config   mockConfig
	rng      *rand.Rand
}

func newStore(cfg mockConfig) *store {
	return &store{
		sessions: make(map[string]*storedSession),
		events:   make([]eventRecord, 0, 128),
		config:   cfg,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *store) appendEvent(sessionID string, eventType string, details string) {
	// Храним только последние события, чтобы не разрасталась память
	s.events = append(s.events, eventRecord{
		SessionID: sessionID,
		Type:      eventType,
		Details:   details,
		Time:      time.Now(),
	})
	if len(s.events) > 300 {
		s.events = s.events[len(s.events)-300:]
	}
}

func (s *store) nextID() string {
	s.counter++
	return "vts_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + strconv.FormatInt(s.counter, 10)
}

func (s *store) createSession(req createSessionRequest) Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	// Если сценарий не передали или он невалидный, берем дефолт из конфига
	scenario := s.resolveScenario(req.Scenario)

	session := &storedSession{
		Session: Session{
			ID:                    s.nextID(),
			ExternalTransactionID: strings.TrimSpace(req.ExternalTransactionID),
			AmountMinor:           req.AmountMinor,
			Currency:              normalizeCurrency(req.Currency),
			Status:                SessionStatusCreated,
			Scenario:              scenario,
			CreatedAt:             now,
			UpdatedAt:             now,
		},
	}
	s.sessions[session.ID] = session
	s.appendEvent(session.ID, "session_created", "status=created")
	return session.Session
}

func (s *store) getSession(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}

	s.applyAutoResultIfDue(session)
	return session.Session, true
}

func (s *store) startSession(id string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return Session{}, errSessionNotFound
	}

	s.applyAutoResultIfDue(session)
	if isFinalStatus(session.Status) {
		return session.Session, nil
	}

	now := time.Now()
	if session.StartedAt == nil {
		session.StartedAt = &now
	}
	session.Status = SessionStatusProcessing
	session.UpdatedAt = now

	delay := time.Duration(s.config.AutoDelayMS) * time.Millisecond
	if delay < 0 {
		delay = 0
	}

	// Планируем автоматический итог после задержки
	outcome := chooseOutcomeForScenario(session.Scenario, s.config, s.rng)
	session.autoOutcome = outcome
	session.autoComplete = now.Add(delay)

	s.appendEvent(session.ID, "session_started", "scheduled="+string(outcome))
	return session.Session, nil
}

func (s *store) finalizeSessionManually(id string, target SessionStatus, reason string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return Session{}, errSessionNotFound
	}

	s.applyAutoResultIfDue(session)
	if isFinalStatus(session.Status) {
		return session.Session, nil
	}

	session.Status = target
	session.Error = reason
	session.UpdatedAt = time.Now()
	session.autoOutcome = ""
	session.autoComplete = time.Time{}

	s.appendEvent(session.ID, "manual_finalize", "status="+string(target))
	return session.Session, nil
}

func (s *store) setSessionScenario(id string, scenario Scenario) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if !ok {
		return Session{}, errSessionNotFound
	}
	s.applyAutoResultIfDue(session)
	if isFinalStatus(session.Status) {
		return session.Session, nil
	}

	session.Scenario = scenario
	session.UpdatedAt = time.Now()
	s.appendEvent(session.ID, "scenario_changed", "scenario="+string(scenario))

	// Если сессия уже в обработке, сразу пересчитываем автозавершение
	if session.Status == SessionStatusProcessing {
		outcome := chooseOutcomeForScenario(session.Scenario, s.config, s.rng)
		session.autoOutcome = outcome
		session.autoComplete = time.Now().Add(time.Duration(s.config.AutoDelayMS) * time.Millisecond)
		s.appendEvent(session.ID, "auto_rescheduled", "scheduled="+string(outcome))
	}

	return session.Session, nil
}

func (s *store) setConfig(req updateConfigRequest) mockConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.DefaultScenario != nil {
		s.config.DefaultScenario = *req.DefaultScenario
	}
	if req.AutoDelayMS != nil {
		s.config.AutoDelayMS = *req.AutoDelayMS
	}
	if req.RandomDeclinePct != nil {
		s.config.RandomDeclinePct = clamp(*req.RandomDeclinePct, 0, 100)
	}
	s.appendEvent("", "config_updated", "defaultScenario="+string(s.config.DefaultScenario))
	return s.config
}

func (s *store) getConfig() mockConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *store) getEvents(limit int) []eventRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	start := len(s.events) - limit
	out := make([]eventRecord, limit)
	copy(out, s.events[start:])
	return out
}

func (s *store) resolveScenario(raw string) Scenario {
	parsed, ok := parseScenario(raw)
	if ok {
		return parsed
	}
	return s.config.DefaultScenario
}

func (s *store) applyAutoResultIfDue(session *storedSession) {
	// Меняем статус только когда пришло время автозавершения
	if session.Status != SessionStatusProcessing {
		return
	}
	if session.autoOutcome == "" || session.autoComplete.IsZero() {
		return
	}
	if time.Now().Before(session.autoComplete) {
		return
	}

	session.Status = session.autoOutcome
	session.UpdatedAt = time.Now()
	if session.Status == SessionStatusDeclined {
		session.Error = "declined by mock scenario"
	}
	if session.Status == SessionStatusTimeout {
		session.Error = "operation timeout in mock scenario"
	}
	s.appendEvent(session.ID, "auto_finalize", "status="+string(session.Status))

	session.autoOutcome = ""
	session.autoComplete = time.Time{}
}

type createSessionRequest struct {
	ExternalTransactionID string `json:"externalTransactionId"`
	AmountMinor           int64  `json:"amountMinor"`
	Currency              string `json:"currency"`
	Scenario              string `json:"scenario"`
}

type updateScenarioRequest struct {
	Scenario string `json:"scenario"`
}

type updateConfigRequest struct {
	DefaultScenario  *Scenario `json:"defaultScenario"`
	AutoDelayMS      *int      `json:"autoDelayMs"`
	RandomDeclinePct *int      `json:"randomDeclinePct"`
}

var errSessionNotFound = &apiError{
	StatusCode: http.StatusNotFound,
	Message:    "session not found",
}

type apiError struct {
	StatusCode int
	Message    string
}

func (e *apiError) Error() string {
	return e.Message
}

func loadMockEnv() {
	// Проверяем несколько путей, чтобы запускать сервис из разных директорий
	candidates := []string{
		"server/mocks/vendotek/.env",
		"mocks/vendotek/.env",
		".env",
	}

	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			log.Printf("mock vendotek: loaded env from %s", path)
			return
		}
	}

	log.Printf("mock vendotek: .env not found, using system environment")
}

func main() {
	loadMockEnv()

	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)

	cfg := mockConfig{
		DefaultScenario:  mustScenarioOrDefault(os.Getenv("VENDOTEK_DEFAULT_SCENARIO"), ScenarioApproved),
		AutoDelayMS:      envIntOrDefault("VENDOTEK_AUTO_DELAY_MS", 1200),
		RandomDeclinePct: clamp(envIntOrDefault("VENDOTEK_RANDOM_DECLINE_PCT", 20), 0, 100),
	}
	mockStore := newStore(cfg)

	// Основные API роуты мок сервиса
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "mock-vendotek",
			"time":    time.Now().Format(time.RFC3339),
			"config":  mockStore.getConfig(),
		})
	})

	r.POST("/sessions", func(c *gin.Context) {
		var req createSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		// Валидация базовых полей при создании сессии
		if req.AmountMinor <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amountMinor must be > 0"})
			return
		}

		if req.Scenario != "" {
			if _, ok := parseScenario(req.Scenario); !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scenario"})
				return
			}
		}
		c.JSON(http.StatusCreated, mockStore.createSession(req))
	})

	r.GET("/sessions/:id", func(c *gin.Context) {
		session, ok := mockStore.getSession(c.Param("id"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusOK, session)
	})

	r.POST("/sessions/:id/start", func(c *gin.Context) {
		session, err := mockStore.startSession(c.Param("id"))
		if err != nil {
			renderError(c, err)
			return
		}
		c.JSON(http.StatusOK, session)
	})

	r.POST("/sessions/:id/approve", func(c *gin.Context) {
		// Принудительно завершаем сессию как успешную
		session, err := mockStore.finalizeSessionManually(c.Param("id"), SessionStatusApproved, "")
		if err != nil {
			renderError(c, err)
			return
		}
		c.JSON(http.StatusOK, session)
	})

	r.POST("/sessions/:id/decline", func(c *gin.Context) {
		// Принудительно завершаем сессию как отклоненную
		session, err := mockStore.finalizeSessionManually(c.Param("id"), SessionStatusDeclined, "declined manually")
		if err != nil {
			renderError(c, err)
			return
		}
		c.JSON(http.StatusOK, session)
	})

	r.POST("/sessions/:id/cancel", func(c *gin.Context) {
		// Принудительно отменяем сессию
		session, err := mockStore.finalizeSessionManually(c.Param("id"), SessionStatusCancelled, "cancelled manually")
		if err != nil {
			renderError(c, err)
			return
		}
		c.JSON(http.StatusOK, session)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("mock vendotek started on :%s", port)
	log.Printf("default scenario=%s, auto delay=%dms, random decline=%d%%", cfg.DefaultScenario, cfg.AutoDelayMS, cfg.RandomDeclinePct)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("mock vendotek failed: %v", err)
	}
}

func renderError(c *gin.Context, err error) {
	// Преобразуем внутренние ошибки в ответ API
	apiErr, ok := err.(*apiError)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(apiErr.StatusCode, gin.H{"error": apiErr.Message})
}

func chooseOutcomeForScenario(scenario Scenario, cfg mockConfig, rng *rand.Rand) SessionStatus {
	// Выбираем итог на основе сценария для тестовых прогонов
	switch scenario {
	case ScenarioApproved:
		return SessionStatusApproved
	case ScenarioDeclined:
		return SessionStatusDeclined
	case ScenarioTimeout:
		return SessionStatusTimeout
	case ScenarioRandom:
		if rng.Intn(100) < cfg.RandomDeclinePct {
			return SessionStatusDeclined
		}
		return SessionStatusApproved
	default:
		return SessionStatusApproved
	}
}

func parseScenario(raw string) (Scenario, bool) {
	// Нормализуем ввод, чтобы принимать значения в разном регистре
	normalized := Scenario(strings.ToLower(strings.TrimSpace(raw)))
	if isKnownScenario(normalized) {
		return normalized, true
	}
	return "", false
}

func isKnownScenario(value Scenario) bool {
	switch value {
	case ScenarioApproved, ScenarioDeclined, ScenarioTimeout, ScenarioRandom:
		return true
	default:
		return false
	}
}

func mustScenarioOrDefault(raw string, fallback Scenario) Scenario {
	if parsed, ok := parseScenario(raw); ok {
		return parsed
	}
	return fallback
}

func normalizeCurrency(raw string) string {
	// По умолчанию используем рубли
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return "RUB"
	}
	return value
}

func isFinalStatus(status SessionStatus) bool {
	switch status {
	case SessionStatusApproved, SessionStatusDeclined, SessionStatusCancelled, SessionStatusTimeout:
		return true
	default:
		return false
	}
}

func envIntOrDefault(name string, fallback int) int {
	// Безопасно читаем число из env и даем fallback при ошибке
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
