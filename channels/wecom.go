package channels

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

type WecomConfig struct {
	Enabled        bool
	CorpID         string
	AgentID        string
	Secret         string
	Token          string
	EncodingAESKey string
	AllowFrom      []string
}

type WecomChannel struct {
	corpID         string
	agentID        string
	secret         string
	token          string
	encodingAESKey string
	allowedChats   []string
	name           string
	running        bool
	accessToken    string
	tokenExpiresAt time.Time
	httpClient     *http.Client
	messageHandler MessageHandler
}

type MessageHandler func(chatID, userID, message string) error

type WecomMessage struct {
	ToUserName   string `json:"ToUserName"`
	FromUserName string `json:"FromUserName"`
	CreateTime   int64  `json:"CreateTime"`
	MsgType      string `json:"MsgType"`
	Content      string `json:"Content"`
	MsgId        string `json:"MsgId"`
	AgentID      string `json:"AgentID"`
}

type WecomTextMessage struct {
	Touser  string           `json:"touser"`
	Msgtype string           `json:"msgtype"`
	Text    WecomTextContent `json:"text"`
	Agentid string           `json:"agentid"`
}

type WecomTextContent struct {
	Content string `json:"content"`
}

type WecomAccessTokenResponse struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type WecomSendMessageResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
	Msgid   string `json:"msgid"`
}

func NewWecomChannel(cfg *WecomConfig) *WecomChannel {
	return &WecomChannel{
		corpID:         cfg.CorpID,
		agentID:        cfg.AgentID,
		secret:         cfg.Secret,
		token:          cfg.Token,
		encodingAESKey: cfg.EncodingAESKey,
		allowedChats:   cfg.AllowFrom,
		name:           "wecom",
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (wc *WecomChannel) Start() error {
	if wc.corpID == "" || wc.secret == "" {
		return fmt.Errorf("wecom corp_id and secret must be configured")
	}

	if err := wc.refreshAccessToken(); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	wc.running = true
	log.Printf("WeCom channel started (CorpID: %s, AgentID: %s)", wc.corpID, wc.agentID)
	return nil
}

func (wc *WecomChannel) Stop() error {
	wc.running = false
	log.Printf("WeCom channel stopped")
	return nil
}

func (wc *WecomChannel) Name() string {
	return wc.name
}

func (wc *WecomChannel) SetMessageHandler(handler MessageHandler) {
	wc.messageHandler = handler
}

func (wc *WecomChannel) Send(chatID, message string) error {
	if !wc.running {
		return fmt.Errorf("wecom channel not running")
	}

	if !wc.isAllowed(chatID) {
		return fmt.Errorf("chat %s not allowed", chatID)
	}

	if err := wc.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	msg := WecomTextMessage{
		Touser:  chatID,
		Msgtype: "text",
		Text: WecomTextContent{
			Content: message,
		},
		Agentid: wc.agentID,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", wc.accessToken)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := wc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	var result WecomSendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Errcode != 0 {
		if result.Errcode == 40014 || result.Errcode == 42001 {
			wc.accessToken = ""
			return wc.Send(chatID, message)
		}
		return fmt.Errorf("wecom API error: %d - %s", result.Errcode, result.Errmsg)
	}

	log.Printf("WeCom message sent successfully to %s, msgid: %s", chatID, result.Msgid)
	return nil
}

func (wc *WecomChannel) refreshAccessToken() error {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", wc.corpID, wc.secret)

	resp, err := wc.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to request access token: %w", err)
	}
	defer resp.Body.Close()

	var result WecomAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode access token response: %w", err)
	}

	if result.Errcode != 0 {
		return fmt.Errorf("wecom API error: %d - %s", result.Errcode, result.Errmsg)
	}

	wc.accessToken = result.AccessToken
	wc.tokenExpiresAt = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	log.Printf("WeCom access token refreshed, expires in %d seconds", result.ExpiresIn)
	return nil
}

func (wc *WecomChannel) ensureValidToken() error {
	if wc.accessToken == "" || time.Now().After(wc.tokenExpiresAt) {
		return wc.refreshAccessToken()
	}
	return nil
}

func (wc *WecomChannel) isAllowed(chatID string) bool {
	if len(wc.allowedChats) == 0 {
		return true
	}
	for _, allowed := range wc.allowedChats {
		if allowed == chatID {
			return true
		}
	}
	return false
}

func (wc *WecomChannel) VerifyURL(signature, timestamp, nonce, echostr string) (string, error) {
	if !wc.validateSignature(signature, timestamp, nonce) {
		return "", fmt.Errorf("invalid signature")
	}

	decrypted, err := wc.decrypt(echostr)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt echostr: %w", err)
	}

	return decrypted, nil
}

func (wc *WecomChannel) HandleCallback(signature, timestamp, nonce, body string) ([]byte, error) {
	if !wc.validateSignature(signature, timestamp, nonce) {
		return nil, fmt.Errorf("invalid signature")
	}

	decrypted, err := wc.decrypt(body)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	var msg WecomMessage
	if err := json.Unmarshal([]byte(decrypted), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	if wc.messageHandler != nil {
		go func() {
			if err := wc.messageHandler(msg.FromUserName, msg.FromUserName, msg.Content); err != nil {
				log.Printf("Error handling WeCom message: %v", err)
			}
		}()
	}

	return []byte("success"), nil
}

func (wc *WecomChannel) validateSignature(signature, timestamp, nonce string) bool {
	arr := []string{wc.token, timestamp, nonce}
	sort.Strings(arr)
	combined := strings.Join(arr, "")

	h := sha1.New()
	h.Write([]byte(combined))
	calculated := fmt.Sprintf("%x", h.Sum(nil))

	return calculated == signature
}

func (wc *WecomChannel) decrypt(encrypted string) (string, error) {
	if len(wc.encodingAESKey) != 43 {
		return "", fmt.Errorf("invalid encoding AES key length")
	}

	aesKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		aesKey[i] = wc.encodingAESKey[i] & 0xFF
	}

	ciphertext := []byte(encrypted + "=")
	for len(ciphertext)%4 != 0 {
		ciphertext = append(ciphertext, '=')
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	decrypted = pkcs7Unpad(decrypted)

	if len(decrypted) < 16 {
		return "", fmt.Errorf("decrypted data too short")
	}

	msgLen := binary.BigEndian.Uint32(decrypted[16:20])
	if int(20+msgLen) > len(decrypted) {
		return "", fmt.Errorf("invalid message length")
	}

	return string(decrypted[20 : 20+msgLen]), nil
}

func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding > aes.BlockSize {
		return data
	}
	return data[:len(data)-padding]
}

func (wc *WecomChannel) SendMarkdown(chatID, content string) error {
	if !wc.running {
		return fmt.Errorf("wecom channel not running")
	}

	if err := wc.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	msg := map[string]interface{}{
		"touser":  chatID,
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
		"agentid": wc.agentID,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", wc.accessToken)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := wc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	var result WecomSendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Errcode != 0 {
		return fmt.Errorf("wecom API error: %d - %s", result.Errcode, result.Errmsg)
	}

	return nil
}

func (wc *WecomChannel) SendCard(chatID, title, description, cardURL string) error {
	if !wc.running {
		return fmt.Errorf("wecom channel not running")
	}

	if err := wc.ensureValidToken(); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	msg := map[string]interface{}{
		"touser":  chatID,
		"msgtype": "textcard",
		"textcard": map[string]string{
			"title":       title,
			"description": description,
			"url":         cardURL,
			"btntxt":      "查看详情",
		},
		"agentid": wc.agentID,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	apiURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", wc.accessToken)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := wc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	var result WecomSendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Errcode != 0 {
		return fmt.Errorf("wecom API error: %d - %s", result.Errcode, result.Errmsg)
	}

	return nil
}
