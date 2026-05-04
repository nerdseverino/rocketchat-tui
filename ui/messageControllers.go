package ui

import (
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
	"github.com/RocketChat/Rocket.Chat.Go.SDK/realtime"
	tea "github.com/charmbracelet/bubbletea"
)

type connectionCheckMsg struct{}
type reconnectedMsg struct{}

func connectionCheckTick() tea.Cmd {
	return tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
		return connectionCheckMsg{}
	})
}

func (m *Model) setupStatusListener(reconnectCh chan struct{}) {
	m.rlClient.AddStatusListener(func(status int) {
		if status == 3 {
			select {
			case reconnectCh <- struct{}{}:
			default:
			}
		}
	})
}

func waitForReconnect(reconnectCh chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-reconnectCh
		return reconnectedMsg{}
	}
}

func (m *Model) handleReconnect() tea.Cmd {
	log.Println("reconnect detected, performing full reconnect")
	return m.fullReconnect()
}

func (m *Model) checkConnection() tea.Cmd {
	if m.rlClient == nil {
		return connectionCheckTick()
	}

	// Usar goroutine com timeout para detectar conexão morta
	done := make(chan error, 1)
	go func() {
		done <- m.rlClient.ConnectionOnline()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Println("connection check failed:", err)
			m.connectionAlive = false
			return tea.Batch(m.fullReconnect(), connectionCheckTick())
		}
	case <-time.After(5 * time.Second):
		log.Println("connection check timeout, forcing full reconnect")
		m.connectionAlive = false
		return tea.Batch(m.fullReconnect(), connectionCheckTick())
	}

	return connectionCheckTick()
}

// fullReconnect cria um novo client DDP, re-loga com token e re-subscreve
func (m *Model) fullReconnect() tea.Cmd {
	serverUrl, err := url.Parse(m.serverUrl)
	if err != nil {
		log.Println("fullReconnect: bad URL:", err)
		return nil
	}

	// Fechar conexão antiga
	if m.rlClient != nil {
		m.rlClient.Close()
	}

	// Novo client
	c, err := realtime.NewClient(serverUrl, false)
	if err != nil {
		log.Println("fullReconnect: connect failed:", err)
		return connectionCheckTick()
	}

	// Re-login com token
	_, err = c.Login(&models.UserCredentials{Token: m.token})
	if err != nil {
		log.Println("fullReconnect: login failed:", err)
		c.Close()
		return connectionCheckTick()
	}

	m.rlClient = c
	m.subscribed = make(map[string]string)

	// Re-subscribe ao canal ativo
	if m.activeChannel.RoomId != "" {
		if err := m.rlClient.SubscribeToMessageStream(&models.Channel{ID: m.activeChannel.RoomId}, m.msgChannel); err != nil {
			log.Println("fullReconnect: subscribe failed:", err)
		} else {
			m.subscribed[m.activeChannel.RoomId] = m.activeChannel.RoomId
		}
	}

	m.connectionAlive = true
	m.setupStatusListener(m.reconnectCh)
	log.Println("fullReconnect: success")
	return waitForReconnect(m.reconnectCh)
}

// It calls the Realtime API function used to send message in the TUI.
func (m *Model) sendMessage(text string) {
	if text != "" {
		channelId := m.activeChannel.RoomId
		retryWithBackoff(2, func() error {
			_, err := m.rlClient.SendMessage(&models.Message{RoomID: channelId, Msg: text})
			return err
		})
	}
}

type historyLoadedMsg struct {
	messages  []models.Message
	timestamp *time.Time
}

type pastMessagesMsg struct {
	messages  []models.Message
	timestamp *time.Time
}

// It calls the Realtime API function to load past message history of a room when the TUI first rendered.
func (m *Model) loadHistory() tea.Cmd {
	channelId := m.activeChannel.RoomId
	return func() tea.Msg {
		var messages []models.Message
		err := retryWithBackoff(3, func() error {
			var e error
			messages, e = m.rlClient.LoadHistory(channelId)
			return e
		})
		if err != nil {
			log.Println(err)
			return historyLoadedMsg{}
		}
		if len(messages) == 0 {
			return historyLoadedMsg{}
		}
		// Reverse order
		for i := len(messages)/2 - 1; i >= 0; i-- {
			opp := len(messages) - 1 - i
			messages[i], messages[opp] = messages[opp], messages[i]
		}
		return historyLoadedMsg{messages: messages, timestamp: messages[0].Timestamp}
	}
}

// It calls the REST API function to fetch more past messages of a romm.
// It is called when user want to load more past message.
// It calls the appropriate API according to the type of channel from public (channel), private (group) and DM
func (m *Model) fetchPastMessages() tea.Cmd {
	today := m.lastMessageTimestamp
	if today == nil {
		return nil
	}
	ts := *today
	channel := &models.Channel{
		ID:    m.activeChannel.RoomId,
		Name:  m.activeChannel.Name,
		Fname: m.activeChannel.DisplayName,
		Type:  m.activeChannel.Type,
	}
	page := &models.Pagination{Count: 50, Offset: len(m.messageHistory)}
	restClient := m.restClient

	return func() tea.Msg {
		var (
			messages []models.Message
			err      error
		)
		switch channel.Type {
		case "c":
			messages, err = restClient.ChannelHistory(channel, true, ts, page)
		case "d":
			messages, err = restClient.DMHistory(channel, true, ts, page)
		default:
			messages, err = restClient.GroupHistory(channel, true, ts, page)
		}
		if err != nil {
			log.Println("fetch past messages error:", err)
			return pastMessagesMsg{}
		}
		if len(messages) == 0 {
			return pastMessagesMsg{}
		}
		for i := len(messages)/2 - 1; i >= 0; i-- {
			opp := len(messages) - 1 - i
			messages[i], messages[opp] = messages[opp], messages[i]
		}
		return pastMessagesMsg{messages: messages, timestamp: messages[0].Timestamp}
	}
}

// It is used for the realtime updation of chat messages in the TUI.
// It return a 'tea.Cmd' which is a function which returns 'tea.Msg' as it triggers the Update function.
// The 'tea.Msg' here returned will be of type models.Message which is catched in TUI Update function and hence TUI is updated with new message.
func (m *Model) waitForIncomingMessage(msgChannel chan models.Message) tea.Cmd {
	return func() tea.Msg {
		return <-msgChannel
	}
}

// It is used for handling sending of message from the TUI and check message before sending.
func (m *Model) handleMessageSending() {
	msg := strings.TrimSpace(m.textInput.Value())
	if msg != "" {
		m.sendMessage(msg)
		m.textInput.Reset()
		return
	}
	m.textInput.Reset()
}
