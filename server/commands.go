package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) executeGrant(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	if len(params) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Usage: `/stick grant @username`",
		}, nil
	}

	username := strings.TrimPrefix(params[0], "@")
	user, err := p.API.GetUserByUsername(username)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("User @%s not found.", username),
		}, nil
	}

	state, err := p.getChannelState(args.ChannelId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to get channel state.",
		}, nil
	}

	state.Speakers[user.Id] = true

	if err := p.setChannelState(args.ChannelId, state); err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to grant speaking privileges.",
		}, nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("@%s has been granted speaking privileges.", username),
	}, nil
}

func (p *Plugin) executeRevoke(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	if len(params) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Usage: `/stick revoke @username`",
		}, nil
	}

	username := strings.TrimPrefix(params[0], "@")
	user, err := p.API.GetUserByUsername(username)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("User @%s not found.", username),
		}, nil
	}

	state, err := p.getChannelState(args.ChannelId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to get channel state.",
		}, nil
	}

	delete(state.Speakers, user.Id)
	delete(state.QASlots, user.Id)

	if err := p.setChannelState(args.ChannelId, state); err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to revoke speaking privileges.",
		}, nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("@%s has had their speaking privileges revoked.", username),
	}, nil
}

func (p *Plugin) executeList(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	state, err := p.getChannelState(args.ChannelId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to get channel state.",
		}, nil
	}

	var speakers []string
	for userID := range state.Speakers {
		user, err := p.API.GetUser(userID)
		if err != nil || user == nil {
			p.API.LogWarn("Failed to get user for speaker list", "user_id", userID, "error", err)
			continue
		}
		speakers = append(speakers, fmt.Sprintf("@%s", user.Username))
	}

	var qaParticipants []string
	for userID, slots := range state.QASlots {
		if slots > 0 {
			user, err := p.API.GetUser(userID)
			if err != nil || user == nil {
				p.API.LogWarn("Failed to get user for Q&A list", "user_id", userID, "error", err)
				continue
			}
			qaParticipants = append(qaParticipants, fmt.Sprintf("@%s (%d slots)", user.Username, slots))
		}
	}

	text := fmt.Sprintf("### Talking Stick Status\n\n**Mode:** %s\n\n", state.Mode)

	if len(speakers) > 0 {
		text += fmt.Sprintf("**Speakers:** %s\n\n", strings.Join(speakers, ", "))
	} else {
		text += "**Speakers:** None\n\n"
	}

	if len(qaParticipants) > 0 {
		text += fmt.Sprintf("**Q&A Participants:** %s", strings.Join(qaParticipants, ", "))
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}, nil
}

func (p *Plugin) executeMode(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	if len(params) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Usage: `/stick mode [open|speakers|qa|locked]`",
		}, nil
	}

	mode := ChannelMode(params[0])
	validModes := map[ChannelMode]bool{
		ModeOpen:         true,
		ModeSpeakersOnly: true,
		ModeQA:           true,
		ModeLocked:       true,
	}

	if !validModes[mode] {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Invalid mode. Options: open, speakers, qa, locked",
		}, nil
	}

	state, err := p.getChannelState(args.ChannelId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to get channel state.",
		}, nil
	}

	state.Mode = mode

	if err := p.setChannelState(args.ChannelId, state); err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to set channel mode.",
		}, nil
	}

	modeDescriptions := map[ChannelMode]string{
		ModeOpen:         "everyone can post",
		ModeSpeakersOnly: "only granted speakers can post",
		ModeQA:           "speakers and Q&A participants can post",
		ModeLocked:       "only administrators can post",
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("Channel mode set to **%s** (%s).", mode, modeDescriptions[mode]),
	}, nil
}

func (p *Plugin) executeQAGrant(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	if len(params) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Usage: `/stick qa-grant @username [count]`",
		}, nil
	}

	username := strings.TrimPrefix(params[0], "@")
	user, err := p.API.GetUserByUsername(username)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("User @%s not found.", username),
		}, nil
	}

	slots := 1
	if len(params) > 1 {
		parsed, err := strconv.Atoi(params[1])
		if err != nil || parsed < 1 {
			return &model.CommandResponse{
				ResponseType: model.CommandResponseTypeEphemeral,
				Text:         "Invalid slot count. Must be a positive number.",
			}, nil
		}
		slots = parsed
	}

	state, err := p.getChannelState(args.ChannelId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to get channel state.",
		}, nil
	}

	state.QASlots[user.Id] = slots

	if err := p.setChannelState(args.ChannelId, state); err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to grant Q&A slots.",
		}, nil
	}

	slotText := "slot"
	if slots > 1 {
		slotText = "slots"
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         fmt.Sprintf("@%s has been granted %d Q&A %s.", username, slots, slotText),
	}, nil
}

func (p *Plugin) settleHelpResponse() *model.CommandResponse {
	help := `### Settle Command

Temporarily suppress agent responses to break doom loops.

**Usage:**
- ` + "`/settle`" + ` - Settle all agents for 20 seconds (default)
- ` + "`/settle 45`" + ` - Settle all agents for 45 seconds
- ` + "`/settle @telos`" + ` - Settle specific agent for 20 seconds
- ` + "`/settle 30 @telos @aurora`" + ` - Settle specific agents for 30 seconds
- ` + "`/settle status`" + ` - Check current settle state

**Notes:**
- Only administrators can use this command
- Maximum settle duration: 300 seconds (5 minutes)
- Responses from settled agents will vanish completely
- Settle expires silently (no announcement)
`

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         help,
	}
}

func (p *Plugin) executeSettle(args *model.CommandArgs, params []string) (*model.CommandResponse, *model.AppError) {
	// Admin check - only admins can use settle
	canBypass, err := p.canBypassTalkingStick(args.UserId, args.ChannelId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to check permissions.",
		}, nil
	}
	if !canBypass {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Only administrators can use the settle command.",
		}, nil
	}

	// Parse params: /settle [seconds] [@user1 @user2...]
	// Examples: /settle, /settle 45, /settle @telos, /settle 30 @telos @aurora

	seconds := 20 // Default 20 seconds
	var targetUsernames []string

	for _, param := range params {
		if strings.HasPrefix(param, "@") {
			// It's a username
			targetUsernames = append(targetUsernames, strings.TrimPrefix(param, "@"))
		} else {
			// Try to parse as seconds
			parsed, err := strconv.Atoi(param)
			if err == nil && parsed > 0 && parsed <= 300 { // Max 5 minutes
				seconds = parsed
			}
		}
	}

	// Get current state
	state, appErr := p.getChannelState(args.ChannelId)
	if appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to get channel state.",
		}, nil
	}

	// Save the current mode so we can restore it later
	previousMode := state.Mode

	// Set settle state
	state.SettleUntil = model.GetMillis() + int64(seconds*1000)
	state.PreviousMode = previousMode

	if len(targetUsernames) == 0 {
		// Settle all - lock the channel
		state.SettleAgents = []string{"all"}
		state.Mode = ModeLocked
	} else {
		// Settle specific users - keep current mode, let MessageWillBePosted handle it
		state.SettleAgents = targetUsernames
	}

	// Save state
	if err := p.setChannelState(args.ChannelId, state); err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to set settle state.",
		}, nil
	}

	// Schedule automatic restore of previous mode
	if len(targetUsernames) == 0 {
		go func() {
			time.Sleep(time.Duration(seconds) * time.Second)
			currentState, err := p.getChannelState(args.ChannelId)
			if err == nil && currentState.SettleUntil > 0 {
				// Restore previous mode
				currentState.Mode = currentState.PreviousMode
				currentState.SettleUntil = 0
				currentState.SettleAgents = []string{}
				p.setChannelState(args.ChannelId, currentState)
			}
		}()
	}

	// Build response message
	issuer, _ := p.API.GetUser(args.UserId)
	issuerUsername := "Someone"
	if issuer != nil {
		issuerUsername = issuer.Username
	}

	var text string
	if len(targetUsernames) == 0 {
		text = fmt.Sprintf("%s has settled the channel for %d seconds", issuerUsername, seconds)
	} else if len(targetUsernames) == 1 {
		text = fmt.Sprintf("%s has settled @%s for %d seconds", issuerUsername, targetUsernames[0], seconds)
	} else {
		text = fmt.Sprintf("%s has settled @%s for %d seconds", issuerUsername, strings.Join(targetUsernames, ", @"), seconds)
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeInChannel,
		Text:         text,
	}, nil
}

func (p *Plugin) executeSettleStatus(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	state, appErr := p.getChannelState(args.ChannelId)
	if appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to get channel state.",
		}, nil
	}

	now := model.GetMillis()

	// Check if settle is active
	if state.SettleUntil == 0 || now >= state.SettleUntil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Channel is not currently settled.",
		}, nil
	}

	// Calculate remaining time
	remainingMs := state.SettleUntil - now
	remainingSeconds := int(remainingMs / 1000)

	var text string
	if len(state.SettleAgents) == 1 && state.SettleAgents[0] == "all" {
		text = fmt.Sprintf("Channel settled for %d more seconds (all agents)", remainingSeconds)
	} else if len(state.SettleAgents) == 1 {
		text = fmt.Sprintf("@%s settled for %d more seconds", state.SettleAgents[0], remainingSeconds)
	} else {
		text = fmt.Sprintf("@%s settled for %d more seconds", strings.Join(state.SettleAgents, ", @"), remainingSeconds)
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}, nil
}
