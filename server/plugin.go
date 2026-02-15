package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex
	configuration     *configuration
}

type configuration struct {
	AllowSystemAdmins  bool
	AllowTeamAdmins    bool
	AllowChannelAdmins bool
	AllowBots          bool
	SuppressionPhrases string
}

type ChannelMode string

const (
	ModeOpen         ChannelMode = "open"
	ModeSpeakersOnly ChannelMode = "speakers"
	ModeQA           ChannelMode = "qa"
	ModeLocked       ChannelMode = "locked"
)

type ChannelState struct {
	Mode         ChannelMode     `json:"mode"`
	Speakers     map[string]bool `json:"speakers"`
	QASlots      map[string]int  `json:"qa_slots"`
	SettleUntil  int64           `json:"settle_until"`  // Unix timestamp in milliseconds
	SettleAgents []string        `json:"settle_agents"` // ["all"] or ["username1", "username2"]
	PreviousMode ChannelMode     `json:"previous_mode"` // Mode to restore after settle expires
}

func (p *Plugin) OnActivate() error {
	p.API.LogInfo("==== TALKING STICK PLUGIN ACTIVATING v0.2.6 ====")
	stickCommand := &model.Command{
		Trigger:          "stick",
		DisplayName:      "Talking Stick",
		Description:      "Manage speaking permissions in channels",
		AutoComplete:     true,
		AutoCompleteDesc: "Manage channel moderation and speaking privileges",
		AutoCompleteHint: "[grant|revoke|list|mode|qa-grant|help]",
	}

	if err := p.API.RegisterCommand(stickCommand); err != nil {
		return fmt.Errorf("failed to register stick command: %w", err)
	}

	settleCommand := &model.Command{
		Trigger:          "settle",
		DisplayName:      "Settle",
		Description:      "Temporarily suppress agent responses (circuit breaker for doom loops)",
		AutoComplete:     true,
		AutoCompleteDesc: "Temporarily quiet agent responses in channel",
		AutoCompleteHint: "[seconds] [@agent1 @agent2...] or 'status'",
	}

	if err := p.API.RegisterCommand(settleCommand); err != nil {
		return fmt.Errorf("failed to register settle command: %w", err)
	}

	p.API.LogInfo("==== TALKING STICK PLUGIN ACTIVATED - MessageWillBePosted hook should now intercept all messages ====")
	return nil
}

func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{
			AllowSystemAdmins:  true,
			AllowTeamAdmins:    true,
			AllowChannelAdmins: true,
			AllowBots:          true,
		}
	}

	return p.configuration
}

func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return fmt.Errorf("failed to load plugin configuration: %w", err)
	}

	p.configurationLock.Lock()
	p.configuration = configuration
	p.configurationLock.Unlock()

	return nil
}

func (p *Plugin) getChannelState(channelID string) (*ChannelState, *model.AppError) {
	key := fmt.Sprintf("channel_%s", channelID)
	data, err := p.API.KVGet(key)
	if err != nil {
		return nil, model.NewAppError("getChannelState", "app.plugin.kv_get.app_error", nil, "", 500)
	}

	if data == nil {
		return &ChannelState{
			Mode:     ModeOpen,
			Speakers: make(map[string]bool),
			QASlots:  make(map[string]int),
		}, nil
	}

	var state ChannelState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, model.NewAppError("getChannelState", "app.plugin.unmarshal.app_error", nil, "", 500)
	}

	if state.Speakers == nil {
		state.Speakers = make(map[string]bool)
	}
	if state.QASlots == nil {
		state.QASlots = make(map[string]int)
	}
	if state.SettleAgents == nil {
		state.SettleAgents = []string{}
	}

	return &state, nil
}

func (p *Plugin) setChannelState(channelID string, state *ChannelState) *model.AppError {
	key := fmt.Sprintf("channel_%s", channelID)
	data, err := json.Marshal(state)
	if err != nil {
		return model.NewAppError("setChannelState", "app.plugin.marshal.app_error", nil, "", 500)
	}

	if err := p.API.KVSet(key, data); err != nil {
		return model.NewAppError("setChannelState", "app.plugin.kv_set.app_error", nil, "", 500)
	}
	return nil
}

func (p *Plugin) canBypassTalkingStick(userID string, channelID string) (bool, error) {
	config := p.getConfiguration()

	user, err := p.API.GetUser(userID)
	if err != nil || user == nil {
		p.API.LogWarn("Failed to get user in bypass check", "user_id", userID, "error", err)
		return false, nil
	}

	if config.AllowBots && user.IsBot {
		return true, nil
	}

	if config.AllowSystemAdmins {
		if strings.Contains(user.Roles, "system_admin") {
			return true, nil
		}
	}

	member, err := p.API.GetChannelMember(channelID, userID)
	if err != nil || member == nil {
		p.API.LogWarn("Failed to get channel member in bypass check", "channel_id", channelID, "user_id", userID, "error", err)
		return false, nil
	}

	if config.AllowChannelAdmins && member.SchemeAdmin {
		return true, nil
	}

	if config.AllowTeamAdmins {
		channel, err := p.API.GetChannel(channelID)
		if err != nil || channel == nil {
			p.API.LogWarn("Failed to get channel in bypass check", "channel_id", channelID, "error", err)
			return false, nil
		}

		teamMember, err := p.API.GetTeamMember(channel.TeamId, userID)
		if err != nil || teamMember == nil {
			p.API.LogWarn("Failed to get team member in bypass check", "team_id", channel.TeamId, "user_id", userID, "error", err)
			return false, nil
		}

		if teamMember.SchemeAdmin {
			return true, nil
		}
	}

	return false, nil
}

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	defer func() {
		if r := recover(); r != nil {
			p.API.LogError("Panic in MessageWillBePosted", "panic", r)
		}
	}()

	if post == nil || post.IsSystemMessage() {
		return post, ""
	}

	// Suppress meta-commentary phrases
	// This allows agents to respond to system prompts, but the responses are filtered out
	p.API.LogInfo("MessageWillBePosted called", "message", post.Message)
	channel, _ := p.API.GetChannel(post.ChannelId)
	channelType := "unknown"
	if channel != nil { channelType = string(channel.Type) }
	p.API.LogInfo("Channel info", "channelId", post.ChannelId, "channelType", channelType)
	config := p.getConfiguration()
	p.API.LogInfo("Checking suppression", "hasConfig", config.SuppressionPhrases != "", "message", post.Message)
	if config.SuppressionPhrases != "" {
		lowerMsg := strings.ToLower(post.Message)
		phrases := strings.Split(config.SuppressionPhrases, "\n")
		for _, phrase := range phrases {
			phrase = strings.TrimSpace(strings.ToLower(phrase))
			if phrase != "" && strings.Contains(lowerMsg, phrase) {
				p.API.LogWarn("SUPPRESSING MESSAGE", "phrase", phrase, "message", post.Message)
				return nil, plugin.DismissPostError // Silently suppress
			}
		}
	}

	// Get state early for both settle and talking stick checks
	state, appErr := p.getChannelState(post.ChannelId)
	if appErr != nil {
		p.API.LogError("Failed to get channel state", "error", appErr.Error())
		return post, ""
	}

	if state == nil {
		return post, ""
	}

	// Check talking stick permissions
	canBypass, _ := p.canBypassTalkingStick(post.UserId, post.ChannelId)
	if canBypass {
		return post, ""
	}

	switch state.Mode {
	case ModeOpen:
		return post, ""

	case ModeSpeakersOnly:
		if state.Speakers[post.UserId] {
			return post, ""
		}
		return nil, "This channel is in speakers-only mode. You do not have speaking privileges."

	case ModeQA:
		if state.Speakers[post.UserId] {
			return post, ""
		}

		slots, hasSlots := state.QASlots[post.UserId]
		if hasSlots && slots > 0 {
			state.QASlots[post.UserId] = slots - 1
			if err := p.setChannelState(post.ChannelId, state); err != nil {
				p.API.LogError("Failed to update Q&A slots", "error", err.Error())
			}
			return post, ""
		}

		return nil, "This channel is in Q&A mode. You do not have a question slot."

	case ModeLocked:
		return nil, "This channel is locked. Only administrators can post."

	default:
		return post, ""
	}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	if len(split) < 1 {
		return p.helpResponse(), nil
	}

	// First word is the command trigger (/stick or /settle)
	trigger := strings.TrimPrefix(split[0], "/")

	// Handle /settle command separately
	if trigger == "settle" {
		if len(split) > 1 {
			if split[1] == "status" {
				return p.executeSettleStatus(args)
			}
			if split[1] == "help" {
				return p.settleHelpResponse(), nil
			}
		}
		return p.executeSettle(args, split[1:])
	}

	// Handle /stick commands
	if len(split) < 2 {
		return p.helpResponse(), nil
	}

	command := split[1]

	switch command {
	case "grant":
		return p.executeGrant(args, split[2:])
	case "revoke":
		return p.executeRevoke(args, split[2:])
	case "list":
		return p.executeList(args)
	case "mode":
		return p.executeMode(args, split[2:])
	case "qa-grant":
		return p.executeQAGrant(args, split[2:])
	case "help":
		return p.helpResponse(), nil
	default:
		return p.helpResponse(), nil
	}
}

func (p *Plugin) helpResponse() *model.CommandResponse {
	help := `### Talking Stick Commands

**Grant/Revoke Speaking Privileges:**
- ` + "`/stick grant @username`" + ` - Grant speaking privileges
- ` + "`/stick revoke @username`" + ` - Revoke speaking privileges
- ` + "`/stick list`" + ` - List current speakers

**Channel Modes:**
- ` + "`/stick mode open`" + ` - Everyone can post (default)
- ` + "`/stick mode speakers`" + ` - Only granted speakers can post
- ` + "`/stick mode qa`" + ` - Speakers + Q&A participants can post
- ` + "`/stick mode locked`" + ` - Only admins can post

**Q&A Mode:**
- ` + "`/stick qa-grant @username [count]`" + ` - Grant question slots (default: 1)

**Help:**
- ` + "`/stick help`" + ` - Show this help message

**Note:** Use ` + "`/settle`" + ` command for circuit breaker functionality (see ` + "`/settle help`" + `)
`

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         help,
	}
}

func main() {
	plugin.ClientMain(&Plugin{})
}
