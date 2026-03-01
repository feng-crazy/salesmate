package channels

import (
	"fmt"
	"salesmate/config"
)

type Channel interface {
	Start() error
	Stop() error
	Name() string
	Send(chatID, message string) error
}

type Manager struct {
	channels map[string]Channel
	config   *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	manager := &Manager{
		channels: make(map[string]Channel),
		config:   cfg,
	}

	manager.initChannels()

	return manager
}

func (cm *Manager) initChannels() {
	if cm.config.Channels.Telegram.Enabled {
		telegram := NewTelegramChannel(cm.config.Channels.Telegram.Token, cm.config.Channels.Telegram.AllowFrom)
		cm.Register(telegram)
	}

	if cm.config.Channels.Discord.Enabled {
		discord := NewDiscordChannel(cm.config.Channels.Discord.Token, cm.config.Channels.Discord.AllowFrom)
		cm.Register(discord)
	}

	if cm.config.Channels.Slack.Enabled {
		slack := NewSlackChannel(cm.config.Channels.Slack.BotToken, cm.config.Channels.Slack.AppToken, cm.config.Channels.Slack.AllowFrom)
		cm.Register(slack)
	}

	if cm.config.Channels.Feishu.Enabled {
		feishu := NewFeishuChannel(
			cm.config.Channels.Feishu.AppID,
			cm.config.Channels.Feishu.AppSecret,
			cm.config.Channels.Feishu.EncryptKey,
			cm.config.Channels.Feishu.Verification,
			cm.config.Channels.Feishu.AllowFrom,
		)
		cm.Register(feishu)
	}

	if cm.config.Channels.Mochat.Enabled {
		mochat := NewMochatChannel(
			cm.config.Channels.Mochat.BaseURL,
			cm.config.Channels.Mochat.ClawToken,
			cm.config.Channels.Mochat.AllowFrom,
		)
		cm.Register(mochat)
	}

	if cm.config.Channels.DingTalk.Enabled {
		dingtalk := NewDingTalkChannel(
			cm.config.Channels.DingTalk.ClientID,
			cm.config.Channels.DingTalk.Secret,
			cm.config.Channels.DingTalk.AllowFrom,
		)
		cm.Register(dingtalk)
	}

	if cm.config.Channels.Email.Enabled {
		email := NewEmailChannel(&cm.config.Channels.Email)
		cm.Register(email)
	}

	if cm.config.Channels.QQ.Enabled {
		qq := NewQQChannel(
			cm.config.Channels.QQ.AppID,
			cm.config.Channels.QQ.Secret,
			cm.config.Channels.QQ.AllowFrom,
		)
		cm.Register(qq)
	}

	if cm.config.Channels.WhatsApp.Enabled {
		waConfig := &WhatsAppConfig{
			Enabled:   cm.config.Channels.WhatsApp.Enabled,
			AllowFrom: cm.config.Channels.WhatsApp.AllowFrom,
		}
		whatsapp := NewWhatsAppChannel(waConfig)
		cm.Register(whatsapp)
	}

	if cm.config.Channels.Wecom.Enabled {
		wecomConfig := &WecomConfig{
			Enabled:        cm.config.Channels.Wecom.Enabled,
			CorpID:         cm.config.Channels.Wecom.CorpID,
			AgentID:        cm.config.Channels.Wecom.AgentID,
			Secret:         cm.config.Channels.Wecom.Secret,
			Token:          cm.config.Channels.Wecom.Token,
			EncodingAESKey: cm.config.Channels.Wecom.EncodingAESKey,
			AllowFrom:      cm.config.Channels.Wecom.AllowFrom,
		}
		wecom := NewWecomChannel(wecomConfig)
		cm.Register(wecom)
	}
}

func (cm *Manager) Register(channel Channel) {
	cm.channels[channel.Name()] = channel
}

func (cm *Manager) Get(name string) (Channel, bool) {
	channel, exists := cm.channels[name]
	return channel, exists
}

func (cm *Manager) StartAll() error {
	for name, channel := range cm.channels {
		if err := channel.Start(); err != nil {
			return fmt.Errorf("failed to start channel %s: %w", name, err)
		}
	}
	return nil
}

func (cm *Manager) StopAll() error {
	var lastErr error
	for name, channel := range cm.channels {
		if err := channel.Stop(); err != nil {
			lastErr = fmt.Errorf("failed to stop channel %s: %w", name, err)
		}
	}
	return lastErr
}

func (cm *Manager) SendToChannel(channelName, chatID, message string) error {
	channel, exists := cm.Get(channelName)
	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	return channel.Send(chatID, message)
}

func (cm *Manager) GetEnabledChannels() []string {
	var enabled []string
	for name := range cm.channels {
		enabled = append(enabled, name)
	}
	return enabled
}
