package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

// UpdateModeGo ...
func UpdateModeGo() {
	log.Printf("Update Mode: %t", updateMode)
	log.Print("!!!!!!! Warning !!!!!!!")
	log.Println("Destination Security Group Will Be Revoke Rule, If Security Group ID Exist.")
	c := askForConfirmation("Do you really want to do it ??")

	if !c {
		fmt.Println("Bye...")
		os.Exit(0)
	}
}

// AlertRestoreMessage ...
func AlertRestoreMessage() {
	log.Print("Restore Mode")
	log.Print("!!!!!!! Warning !!!!!!!")
	log.Print("Security Group Will Be Revoke All Rule, And Restore Rules From File.")

	c := askForConfirmation("Are you sure ??")

	if !c {
		fmt.Println("Bye...")
		os.Exit(0)
	}
}

// AlertCreateMessage ...
func AlertCreateMessage() {
	log.Print("Create Mode")
	log.Print("This Tools Will Create Security Group Rules.")

	c := askForConfirmation("Are you sure ??")

	if !c {
		fmt.Println("Bye...")
		os.Exit(0)
	}
}
