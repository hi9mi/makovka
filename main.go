package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	token              string
	floodChannelID     string
	defaultDelay       = time.Second * 24
	customMessageDelay time.Duration
	adminRoleID        string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	token = os.Getenv("DISCORD_BOT_TOKEN")
	floodChannelID = os.Getenv("DISCORD_FLOOD_CHANNEL_ID")
	adminRoleID = os.Getenv("DISCORD_ADMIN_ROLE")
}

func main() {
	if token == "" || floodChannelID == "" {
		fmt.Println("Environment variables DISCORD_BOT_TOKEN or DISCORD_FLOOD_CHANNEL_ID not set")
		return
	}

	dg, err := discordgo.New(fmt.Sprintf("Bot %s", token))
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running. Press CTRL+C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	handleCommand(s, m)
	if m.ChannelID == floodChannelID {
		handleMessageDeletion(s, m)
	}
}

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if isBotAuthorMessage(s, m) {
		return
	}

	const prefix = "!"

	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	content := strings.TrimPrefix(m.Content, prefix)
	args := strings.Fields(content)

	if len(args) == 0 {
		return
	}

	command := args[0]
	args = args[1:]

	switch command {
	case "ping":
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	case "setdelay":
		handleSetMessageDelay(s, m, args)
	default:
		s.ChannelMessageSendReply(m.ChannelID, "Unknown command", &discordgo.MessageReference{
			MessageID: m.ID,
			ChannelID: m.ChannelID,
			GuildID:   m.GuildID,
		})
	}

}

func setMessageDelay(s *discordgo.Session, delay time.Duration) {
	customMessageDelay = delay
	s.ChannelMessageSend(floodChannelID, fmt.Sprintf("Message deletion delay set to %s", delay))
}

func handleSetMessageDelay(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if !isUserAdmin(s, m) {
		s.ChannelMessageSendReply(m.ChannelID, "You must be an admin to use this command", &discordgo.MessageReference{
			MessageID: m.ID,
			ChannelID: m.ChannelID,
			GuildID:   m.GuildID,
		})
	}
	if len(args) == 0 {
		s.ChannelMessageSendReply(m.ChannelID, "Usage: !setdelay <time>", &discordgo.MessageReference{
			MessageID: m.ID,
			ChannelID: m.ChannelID,
			GuildID:   m.GuildID,
		})
		return
	}

	delay, err := time.ParseDuration(args[0])
	if err != nil {
		fmt.Println("Failed to parse duration:", err)
		s.ChannelMessageSendReply(m.ChannelID, "Invalid time format. Use something like '10m', '1h', etc.", &discordgo.MessageReference{
			MessageID: m.ID,
			ChannelID: m.ChannelID,
			GuildID:   m.GuildID,
		})
		return
	}
	setMessageDelay(s, delay)
}

func handleMessageDeletion(s *discordgo.Session, m *discordgo.MessageCreate) {
	delay := getMessageDelay()

	go func() {
		time.Sleep(delay)
		err := s.ChannelMessageDelete(m.ChannelID, m.ID)
		if err != nil {
			fmt.Println("Failed to delete message:", err)
		}
	}()
}

func getMessageDelay() time.Duration {
	if customMessageDelay > 0 {
		fmt.Println("Using custom message delay")
		return customMessageDelay
	}
	return defaultDelay
}

func isUserAdmin(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		fmt.Println("Error getting guild member,", err)
		return false
	}

	for _, role := range member.Roles {
		if role == adminRoleID {
			return true
		}
	}
	return false
}

func isBotAuthorMessage(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	return m.Author.ID == s.State.User.ID
}
