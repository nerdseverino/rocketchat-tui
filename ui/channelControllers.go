package ui

import (
	"log"
	"sort"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// It is used to change channel in the TUI, uses index of active channel in the channel list to set active channel in state.
// Active channel is subscribed to the realtime message channel to receive messages through message channel.
// Past message history of the active channel is loaded.
func (m *Model) changeSelectedChannel(index int) tea.Cmd {
	if index < 0 || index >= len(m.subscriptionList) {
		return nil
	}
	m.activeChannel = m.subscriptionList[index]

	m.messageHistory = []models.Message{}
	m.unreadCount[m.activeChannel.RoomId] = 0
	m.membersLoadedForRoom = ""

	if _, ok := m.subscribed[m.activeChannel.RoomId]; !ok {
		if err := m.rlClient.SubscribeToMessageStream(&models.Channel{ID: m.activeChannel.RoomId}, m.msgChannel); err != nil {
			log.Println(err)
		}

		m.subscribed[m.activeChannel.RoomId] = m.activeChannel.RoomId
	}

	return tea.Batch(m.loadHistory(), m.setSlashCommandsList())
}

// It is used to get list of all the channels in which user is subscribed.
// All the subscribed channels, groups and DMs are stored in state in subscriptions list.
func (m *Model) getSubscriptions() error {
	subscriptions, err := m.rlClient.GetChannelSubscriptions()
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		if sub.Open && sub.Name != "" {
			m.subscriptionList = append(m.subscriptionList, sub)
		}
	}
	// Canais com unread/alert primeiro, depois por nome
	sort.Slice(m.subscriptionList, func(i, j int) bool {
		if m.subscriptionList[i].Alert != m.subscriptionList[j].Alert {
			return m.subscriptionList[i].Alert
		}
		if m.subscriptionList[i].Unread != m.subscriptionList[j].Unread {
			return m.subscriptionList[i].Unread > m.subscriptionList[j].Unread
		}
		return m.subscriptionList[i].Name < m.subscriptionList[j].Name
	})
	return nil
}

// It is used to set channels in the TUI channel list.
// To update TUI with the updated channels list it will return a tea.Cmd.
// It will be called once after user login and active channel is set to first channel in subscribed channel list.
func (m *Model) setChannelsInUiList() tea.Cmd {
	var items []list.Item
	for _, sub := range m.subscriptionList {
		if sub.Open && sub.Name != "" {
			items = append(items, ChannelsItem(sub))
		}
	}
	channelCmd := m.channelList.SetItems(items)
	if len(m.subscriptionList) > 0 {
		m.activeChannel = m.subscriptionList[0]
	}
	return channelCmd
}
