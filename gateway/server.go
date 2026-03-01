package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"salesmate/config"
	"salesmate/sales_agent"
	"salesmate/security"
)

// Server represents the HTTP gateway server
type Server struct {
	config          *config.Config
	salesLoop       *sales_agent.SalesLoop
	safetySystem    *security.SafetySystem
	followUpEngine  *FollowUpEngine
	server          *http.Server
	mu              sync.RWMutex
	running         bool
	requestCount    int64
	startTime       time.Time
}

// GatewayConfig holds gateway server configuration
type GatewayConfig struct {
	Port            int
	HeartbeatInt    int
	HeartbeatEnable bool
	StrictMode      bool
	AutoReply       bool
}

// NewServer creates a new gateway server
func NewServer(cfg *config.Config, salesLoop *sales_agent.SalesLoop) *Server {
	return &Server{
		config:       cfg,
		salesLoop:    salesLoop,
		safetySystem: security.NewSafetySystem(0.7),
		startTime:    time.Now(),
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	port := s.config.Gateway.Port
	if port == 0 {
		port = 18790
	}

	mux := http.NewServeMux()

	// Health and status endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/status", s.handleStatus)

	// Channel webhooks
	mux.HandleFunc("/webhook/feishu", s.handleFeishuWebhook)
	mux.HandleFunc("/webhook/dingtalk", s.handleDingTalkWebhook)
	mux.HandleFunc("/webhook/wecom", s.handleWecomWebhook)
	mux.HandleFunc("/webhook/telegram", s.handleTelegramWebhook)

	// API endpoints
	mux.HandleFunc("/api/v1/chat", s.handleChat)
	mux.HandleFunc("/api/v1/session/", s.handleSession)
	mux.HandleFunc("/api/v1/pipeline", s.handlePipeline)
	mux.HandleFunc("/api/v1/kb", s.handleKnowledgeBase)

	// Admin endpoints
	mux.HandleFunc("/admin/guardrails", s.handleGuardrails)
	mux.HandleFunc("/admin/confidence", s.handleConfidence)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.running = true
	s.mu.Unlock()

	log.Printf("🚀 Gateway server starting on port %d", port)

	// Start follow-up engine
	s.followUpEngine = NewFollowUpEngine(s.salesLoop, s.safetySystem)
	go s.followUpEngine.Start(ctx)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for server error or context cancellation
	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		return s.Stop(context.Background())
	}
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	log.Println("Stopping gateway server...")

	if s.followUpEngine != nil {
		s.followUpEngine.Stop()
	}

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
	}

	s.running = false
	log.Println("Gateway server stopped")
	return nil
}

// HTTP Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(s.startTime).String(),
	}
	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := map[string]interface{}{
		"status":       "running",
		"uptime":       time.Since(s.startTime).String(),
		"requestCount": s.requestCount,
		"startTime":    s.startTime.Format(time.RFC3339),
		"config": map[string]interface{}{
			"port":       s.config.Gateway.Port,
			"heartbeat":  s.config.Gateway.Heartbeat.Enabled,
			"strictMode": s.safetySystem != nil,
		},
	}
	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleFeishuWebhook(w http.ResponseWriter, r *http.Request) {
	s.requestCount++

	var payload FeishuWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Process the message
	sessionID := fmt.Sprintf("feishu:%s", payload.ChatID)
	response, err := s.processMessage(r.Context(), sessionID, payload.Message, payload.UserID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Send response
	respPayload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": response,
		},
	}
	s.writeJSON(w, http.StatusOK, respPayload)
}

func (s *Server) handleDingTalkWebhook(w http.ResponseWriter, r *http.Request) {
	s.requestCount++

	var payload DingTalkWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	sessionID := fmt.Sprintf("dingtalk:%s", payload.ChatID)
	response, err := s.processMessage(r.Context(), sessionID, payload.Message, payload.UserID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": response})
}

func (s *Server) handleWecomWebhook(w http.ResponseWriter, r *http.Request) {
	s.requestCount++

	var payload WecomWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	sessionID := fmt.Sprintf("wecom:%s", payload.FromUserName)
	response, err := s.processMessage(r.Context(), sessionID, payload.Content, payload.FromUserName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeXML(w, http.StatusOK, WecomResponse{
		ToUserName:   payload.FromUserName,
		FromUserName: payload.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      response,
	})
}

func (s *Server) handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	s.requestCount++

	var payload TelegramWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	sessionID := fmt.Sprintf("telegram:%d", payload.Message.Chat.ID)
	response, err := s.processMessage(r.Context(), sessionID, payload.Message.Text, strconv.FormatInt(payload.Message.From.ID, 10))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"method": "sendMessage",
		"chat_id": payload.Message.Chat.ID,
		"text": response,
	})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.requestCount++

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	response, err := s.processMessage(r.Context(), req.SessionID, req.Message, req.UserID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, ChatResponse{
		Response:  response,
		SessionID: req.SessionID,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/session/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		s.writeError(w, http.StatusBadRequest, "Session ID required")
		return
	}
	sessionID := parts[0]

	switch r.Method {
	case http.MethodGet:
		// Get session info
		stage := s.salesLoop.GetCurrentStage(sessionID)
		confidence := s.salesLoop.GetConfidenceScore(sessionID)
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"sessionId":   sessionID,
			"stage":       stage,
			"confidence":  confidence,
			"lastUpdate":  time.Now().Format(time.RFC3339),
		})
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handlePipeline(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get pipeline report
		kb := s.salesLoop.GetSalesKnowledgeBase()
		products := kb.GetAllProducts()
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"products": products,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleKnowledgeBase(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query().Get("q")
		if query == "" {
			s.writeError(w, http.StatusBadRequest, "Query parameter 'q' required")
			return
		}

		kb := s.salesLoop.GetSalesKnowledgeBase()
		products := kb.QueryProduct(query)
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"query":    query,
			"results":  products,
		})
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleGuardrails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get guardrail status
		violations := s.safetySystem.GetGuardrailManager().GetViolations()
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"violations": violations,
			"strictMode": s.safetySystem != nil,
		})
	case http.MethodPost:
		// Update guardrail settings
		var req struct {
			StrictMode *bool `json:"strictMode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}
		if req.StrictMode != nil {
			s.safetySystem.GetGuardrailManager().SetStrictMode(*req.StrictMode)
		}
		s.writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleConfidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		s.writeError(w, http.StatusBadRequest, "Session parameter required")
		return
	}

	confidence := s.salesLoop.GetConfidenceScore(sessionID)
	stage := s.salesLoop.GetCurrentStage(sessionID)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"sessionId":   sessionID,
		"confidence":  confidence,
		"stage":       stage,
	})
}

// processMessage handles incoming messages with safety checks
func (s *Server) processMessage(ctx context.Context, sessionID, message, userID string) (string, error) {
	// Check safety system for incoming message
	if s.safetySystem != nil {
		safetyResult := s.safetySystem.Check(sessionID, message)
		if safetyResult.SuggestedAction == "human_takeover" {
			return "I'm connecting you with a human representative who can better assist you. Please hold.", nil
		}
	}

	// Process through sales loop
	response, err := s.salesLoop.ProcessSalesMessage(message, sessionID)
	if err != nil {
		return "", err
	}

	// Check safety system for outgoing response
	if s.safetySystem != nil {
		replyCheck := s.safetySystem.CheckReply(sessionID, response)
		if !replyCheck.AllowAutoReply {
			// Response violates guardrails, modify or flag for review
			return "Let me verify that information and get back to you shortly.", nil
		}
	}

	return response, nil
}

// Helper methods

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) writeXML(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	// XML encoding would go here
	fmt.Fprintf(w, "%v", data)
}

// Middleware

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

// Request/Response types

type FeishuWebhookPayload struct {
	ChatID   string `json:"chat_id"`
	UserID   string `json:"user_id"`
	Message  string `json:"message"`
	MsgType  string `json:"msg_type"`
}

type DingTalkWebhookPayload struct {
	ChatID  string `json:"chat_id"`
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type WecomWebhookPayload struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	MsgId        int64  `xml:"MsgId"`
}

type WecomResponse struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
}

type TelegramWebhookPayload struct {
	Message struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

type ChatRequest struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
	UserID    string `json:"userId"`
}

type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"sessionId"`
	Timestamp string `json:"timestamp"`
}