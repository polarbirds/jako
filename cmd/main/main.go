package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"github.com/polarbirds/jako/pkg/command"
	"github.com/polarbirds/jako/internal/mimic"
)

var (
	Token string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	createData(dg)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.Contains(m.Content, "!") {
		source, args, err := command.GetCommand(m.Content)
		if err != nil {
			log.Error(err)
			s.UpdateStatus(0, err.Error())
			return
		}

		var discErr error

		switch strings.ToLower(source) {
		case "mimic":
			if len(args) < 1 {
				s.UpdateStatus(0, "missing target")
				return
			}
			target := args[0]
			s.ChannelMessageSend(m.ChannelID, mimic.Generate(target))
		}

		if err != nil {
			log.Error(err)
			s.UpdateStatus(0, err.Error())
		}

		if discErr != nil {
			log.Error(discErr)
		}
	} else {
		mimic.Build(m.Message)
	}
}

func createData(s *discordgo.Session) {
	s.UpdateStatus(0, "Building data...")
	for _, guild := range s.State.Guilds {
		log.Infof("Parsing guild %s: %s", guild.Name, guild.ID)

		channels, err := s.GuildChannels(guild.ID)
		if err != nil {
			log.Fatal(err)
			return
		}

		for _, v := range channels {
			if v.Type != discordgo.ChannelTypeGuildText {
				continue
			}

			log.Infof("name:%s id:%s", v.Name, v.ID)

			msgs := getMessagesFromChannel(s, *v)

			for _, m := range msgs {
				mimic.Build(m)
			}
			log.Infof("%d messages gotten", len(msgs))
		}
	}

	s.UpdateStatus(0, "Finished building data")
}

func getMessagesFromChannel(s *discordgo.Session, channel discordgo.Channel) []*discordgo.Message {
	beforeID := channel.LastMessageID
	var msgs []*discordgo.Message
	var failedAttempts = 0
	for {
		m, err := s.ChannelMessages(channel.ID, 100, beforeID, "", "")
		if err != nil {
			log.Fatal(err)
			if failedAttempts > 10 {
				log.Error(err)
				break
			} else {
				failedAttempts++
				continue
			}
		}

		if len(m) < 1 {
			break
		}

		msgs = append(msgs, m...)
		beforeID = m[len(m)-1].ID
		failedAttempts = 0
	}
	return msgs
}
