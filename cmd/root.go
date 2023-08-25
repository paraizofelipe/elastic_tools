package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/paraizofelipe/elastic_tools/internal/actions"
	"github.com/paraizofelipe/elastic_tools/internal/config"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const APP_NAME = "elastic_tools"

type Config struct {
	Elastic []string `toml:"elastic"`
}

func NewRootCommand(setup *config.ConfigFile) *cli.App {
	app := cli.NewApp()
	app.Name = APP_NAME
	app.Usage = "Elasticsearch Tools CLI"
	app.Version = "1.0.0"

	flags := []cli.Flag{
		altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name:    "elastic",
			Aliases: []string{"e"},
			Value:   cli.NewStringSlice("http://localhost:9200"),
		}),
		&cli.StringFlag{
			Name:       "config-file",
			Aliases:    []string{"f"},
			Value:      fmt.Sprintf("%s/.config/elastic_tools/config.toml", os.Getenv("HOME")),
			HasBeenSet: true,
		},
	}

	app.EnableBashCompletion = true
	app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewTomlSourceFromFlagFunc("config-file"))

	esNodes := setup.Elastic
	esClient, err := actions.CreateClient(esNodes)
	if err != nil {
		log.Fatalf("Error to create client: %s", err)
	}

	app.Flags = flags
	app.Commands = []*cli.Command{
		NewIndexCommand(esClient),
		NewSearchCommand(esClient),
		NewAliasCommand(esClient),
		NewCatCommand(esClient),
	}

	app.CommandNotFound = func(ctx *cli.Context, in string) {
		fmt.Printf("Ops, command %s unknown\n", in)
	}

	return app
}

func parseCLIArguments(input string) []string {
	arguments := strings.Fields(input)
	if len(arguments) == 0 {
		return []string{}
	}
	return append([]string{APP_NAME}, arguments...)
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func NewPrompt(app *cli.App) (pt *prompt.Prompt) {
	options := LoadOptions(app.Commands)

	pt = prompt.New(
		func(in string) {
			if in == "exit" {
				fmt.Println("Exiting REPL...")
				os.Exit(0)
			}

			cliArguments := parseCLIArguments(in)
			if len(cliArguments) > 0 {
				err := app.Run(cliArguments)
				if err != nil {
					fmt.Println(err)
				}
			}
		},
		func(d prompt.Document) (suggest []prompt.Suggest) {
			if d.TextBeforeCursor() == "" {
				return []prompt.Suggest{}
			}
			args := strings.Split(d.TextBeforeCursor(), " ")
			w := d.GetWordBeforeCursor()

			if len(args) < 2 {
				for _, cmd := range app.Commands {
					suggest = append(suggest, prompt.Suggest{Text: cmd.Name, Description: cmd.Usage})
				}
				return prompt.FilterHasPrefix(suggest, w, true)
			}

			for _, arg := range args {
				if strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "'") {
					continue
				}
				if aux, ok := options[arg]; ok {
					suggest = aux
				}
			}

			return prompt.FilterHasPrefix(suggest, w, true)
		},
		prompt.OptionPrefix(">>> "),
		prompt.OptionTitle("Interactive CLI"),
	)
	return
}

func Execute() {

	config, err := config.Load()
	if err != nil {
		log.Fatalf("Error while loading configuration file: %s", err)
	}

	app := NewRootCommand(config)
	pt := NewPrompt(app)
	pt.Run()
}
