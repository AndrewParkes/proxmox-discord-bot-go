package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/caarlos0/env/v6"
)

/*
	Discord Slash commands examples: https://github.com/bwmarrin/discordgo/tree/master/examples/slash_commands
	Proxmox control: https://github.com/dragse/proxmox-api-go
*/

type (
	app struct {
		d *discordgo.Session
		cfg
	}
	cfg struct {
		Proxmax
		DiscordConf
	}
	Proxmax struct {
		hostname string `env:"VAULT_HOST"`
		username string `env:"USERNAME"`
		password string `env:"PASSWORD"`
		node     string `env:"NODE"`
		vm_id    string `env:"VM_ID"`
	}

	DiscordConf struct {
		Token          string `env:"DISCORD_TOKEN"`
		GuildID        string `env:"GUILD_ID"`
		RemoveCommands string `env:"REMOVE_COMMANDS"`
	}
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "list",
			Description: "List servers and status",
		},
		{
			Name:        "start",
			Description: "Command for demonstrating options",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "server",
					Description: "String option",
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"list": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "some server data",
				},
			})
		},
		"start": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options

			msgformat := "Started Server: "
			for _, opt := range options {
				msgformat += "\n" + opt.StringValue()
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msgformat,
				},
			})
		},
	}
)

func (a *app) init() (err error) {
	a.d, err = discordgo.New("Bot " + a.DiscordConf.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	a.d.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	/*session := client.ProxmoxSession{
		Hostname:  a.hostname,
		Username:  a.username,
		Token:     a.password,
		VerifySSL: false,
	}

	proxClient := client.NewProxmoxClient()
	err = proxClient.AddSession(&session)

	if err != nil {
		log.Fatal(err)
	}*/

	return
}

func (a *app) run() (err error) {
	a.d.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	// In this example, we only care about receiving message events.
	a.d.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = a.d.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := a.d.ApplicationCommandCreate(a.d.State.User.ID, a.DiscordConf.GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer a.d.Close()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	a.d.Close()
	return

}

func main() {
	fmt.Println("starting")

	a := &app{}

	// Load config
	if err := env.Parse(&a.cfg); err != nil {
		os.Exit(1)
	}

	if err := a.init(); err != nil {
		os.Exit(1)
	}

	if err := a.run(); err != nil {
		os.Exit(1)
	}

	fmt.Println("stopped")
}
