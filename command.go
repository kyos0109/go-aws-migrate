package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

// AWSSync ...
type AWSSync struct {
	sourceSGLists []ec2.SecurityGroup
	perfixListMap map[string]*PerfixList
	tagsConfig    []Tag
}

var (
	updateMode bool
	sourceSGID string
	yamlConfig *YamlConfig
)

// CommnadRun ...
func CommnadRun() {
	app := &cli.App{
		Name:    "AWS Migrate Tools",
		Version: "0.6",
		Usage:   "Command Line",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "config.yaml",
				Usage:   "Load configuration from `FILE`",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "SecurityGroup",
				Aliases: []string{"sg"},
				Usage:   "Security Groups Migrate",
				Action:  handelSG,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "update",
						Aliases: []string{"u"},
						Usage:   "Security Groups Sync Update Mode",
					},
					&cli.BoolFlag{
						Name:  "DontTouchThisButton",
						Usage: "Clean Destination Security Group Rule, Don't Try It.",
					},
					&cli.StringFlag{
						Name:  "sid",
						Usage: "Just SYNC This Security Group ID (Experiment).",
					},
					&cli.BoolFlag{
						Name:  "src-export",
						Usage: "Export Source Security Group To File.",
					},
					&cli.BoolFlag{
						Name:  "dst-export",
						Usage: "Export Destination Security Group To File.",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output File Location.",
					},
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Restore File Location.",
					},
					&cli.BoolFlag{
						Name:   "src-restore",
						Usage:  "Restore Source Security Group From File.",
						Hidden: true,
					},
					&cli.BoolFlag{
						Name:   "dst-restore",
						Usage:  "Restore Destination Security Group From File.",
						Hidden: true,
					},
					&cli.BoolFlag{
						Name:    "terraform-export",
						Aliases: []string{"tf"},
						Usage:   "Export terraform to file, has to be used with export args.",
					},
					&cli.BoolFlag{
						Name:  "diff",
						Usage: "Compare source and destination security group.",
					},
				},
			},
			{
				Name:    "Route53",
				Aliases: []string{"r53"},
				Usage:   "Route53 Migrate",
				Action:  handelR53,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "src-export",
						Usage: "Export Source Security Group To File.",
					},
					&cli.BoolFlag{
						Name:  "dst-export",
						Usage: "Export Destination Security Group To File.",
					},
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output File Location.",
					},
					&cli.StringFlag{
						Name:    "file",
						Aliases: []string{"f"},
						Usage:   "Restore File Location.",
					},
					&cli.BoolFlag{
						Name:   "src-restore",
						Usage:  "Restore Source Security Group From File.",
						Hidden: true,
					},
					&cli.BoolFlag{
						Name:   "dst-restore",
						Usage:  "Restore Destination Security Group From File.",
						Hidden: true,
					},
					&cli.BoolFlag{
						Name:    "terraform-export",
						Aliases: []string{"tf"},
						Usage:   "Export terraform to file, has to be used with export args.",
					},
				},
			},
			{
				Name:    "VPC",
				Aliases: []string{"vpc"},
				Usage:   "VPC Migrate",
				Action:  handelVPC,
			},
			{
				Name:    "Subnet",
				Aliases: []string{"sub"},
				Usage:   "Subnet Migrate",
				Action:  handelSubnet,
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.EnableBashCompletion = true

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getYamlConfig(configPath string) error {
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &yamlConfig)
	if err != nil {
		fmt.Println("Unmarshal:", err)
		return err
	}

	return nil
}

func handelSG(c *cli.Context) error {
	err := getYamlConfig(c.String("config"))
	if err != nil {
		return err
	}

	updateMode = c.Bool("update")
	sourceSGID = c.String("sid")

	switch {
	case c.Bool("src-export"):
		ExportSecurityGroupRule(&yamlConfig.Setting.Source, c.String("output"), c.Bool("terraform-export"), &yamlConfig.Setting.Tags)
	case c.Bool("dst-export"):
		ExportSecurityGroupRule(&yamlConfig.Setting.Destination, c.String("output"), c.Bool("terraform-export"), &yamlConfig.Setting.Tags)
	case c.Bool("src-restore"):
		AlertRestoreMessage()
		RestoreSecurityGroupRule(&yamlConfig.Setting.Source, c.String("file"))
	case c.Bool("dst-restore"):
		AlertRestoreMessage()
		RestoreSecurityGroupRule(&yamlConfig.Setting.Destination, c.String("file"))
	case updateMode:
		UpdateModeGo()
		SecurityGroupSyncGO(&yamlConfig.Setting)
	case c.Bool("DontTouchThisButton"):
		CleanSecurityGroupRule(&yamlConfig.Setting.Destination)
	case c.Bool("diff"):
		DiffSecurityGroup(&yamlConfig.Setting)
	default:
		AlertCreateMessage()
		SecurityGroupSyncGO(&yamlConfig.Setting)
	}

	return nil
}

func handelR53(c *cli.Context) error {
	err := getYamlConfig(c.String("config"))
	if err != nil {
		return err
	}

	switch {
	case c.Bool("src-export"):
		ExportRoute53Record(&yamlConfig.Setting.Source, c.String("output"), c.Bool("terraform-export"), &yamlConfig.Setting.Tags)
	case c.Bool("dst-export"):
		ExportRoute53Record(&yamlConfig.Setting.Destination, c.String("output"), c.Bool("terraform-export"), &yamlConfig.Setting.Tags)
	default:
		cc := askForConfirmation("Do you really want to do it ??")

		if !cc {
			fmt.Println("Bye...")
			os.Exit(0)
		}

		Route53SyncGO(&yamlConfig.Setting)
	}

	return nil
}

func handelVPC(c *cli.Context) error {
	err := getYamlConfig(c.String("config"))
	if err != nil {
		return err
	}

	cc := askForConfirmation("Do you really want to do it ??")

	if !cc {
		fmt.Println("Bye...")
		os.Exit(0)
	}

	VPCSyncGO(&yamlConfig.Setting)

	return nil
}

func handelSubnet(c *cli.Context) error {
	err := getYamlConfig(c.String("config"))
	if err != nil {
		return err
	}

	cc := askForConfirmation("Do you really want to do it ??")

	if !cc {
		fmt.Println("Bye...")
		os.Exit(0)
	}

	SubnetSyncGO(&yamlConfig.Setting)

	return nil
}
