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
	Token              string
	floodChannelID     string
	defaultDelay       = time.Hour * 24
	customMessageDelay time.Duration
	adminRoleID        string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	Token = os.Getenv("DISCORD_BOT_TOKEN")
	floodChannelID = os.Getenv("DISCORD_FLOOD_CHANNEL_ID")
	adminRoleID = os.Getenv("ADMIN_ROLE_ID")
}

func main() {
	if Token == "" || floodChannelID == "" {
		fmt.Println("Environment variables DISCORD_BOT_TOKEN or DISCORD_FLOOD_CHANNEL_ID not set")
		return
	}

	dg, err := discordgo.New(fmt.Sprintf("Bot %s", Token))
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
	if m.Author.ID == s.State.User.ID {
		return
	}

	handleCommand(s, m)
	if m.ChannelID == floodChannelID {
		handleMessageDeletion(s, m)
	}
}

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	content := m.Content
	channelID := m.ChannelID
	fmt.Println(content)
	fmt.Print(content, "Hello")
	if strings.HasPrefix(content, "!setdelay") {
		parts := strings.Split(content, " ")
		fmt.Println(parts)
		if len(parts) != 2 {
			s.ChannelMessageSend(channelID, "Usage: !setdelay <time>")
			return
		}

		delayDuration, err := time.ParseDuration(parts[1])
		if err != nil {
			s.ChannelMessageSend(channelID, "Invalid time format. Use something like '10m', '1h', etc.")
			return
		}

		if !isUserAdmin(s, m) {
			s.ChannelMessageSend(channelID, "You do not have permission to set the delay.")
			return
		}

		setMessageDelay(s, delayDuration)
	}
}

func setMessageDelay(s *discordgo.Session, delay time.Duration) {
	customMessageDelay = delay
	s.ChannelMessageSend(floodChannelID, fmt.Sprintf("Message deletion delay set to %s", delay))
}

func handleMessageDeletion(s *discordgo.Session, m *discordgo.MessageCreate) {
	delay := getMessageDelay()
	go func(messageID string) {
		time.Sleep(delay)
		err := s.ChannelMessageDelete(floodChannelID, messageID)
		if err != nil {
			fmt.Println("Failed to delete message:", err)
		}
	}(m.ID)
}

func getMessageDelay() time.Duration {
	if customMessageDelay > 0 {
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
