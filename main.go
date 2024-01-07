package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type monitor struct {
	Win string
	No  string
	Mac string
}

var mappings = map[string]monitor{
	"asus": monitor{
		Win: "0x0f",
		Mac: "0x11",
		No:  "2",
	},
	"aoc": monitor{
		Win: "0x0f",
		Mac: "0x10",
		No:  "1",
	},
}

// 60(0F 10 11 12 )
func main() {
	asus := mappings["asus"]
	aoc := mappings["aoc"]

	cmd := exec.Command("./winddcutil.exe", "getvcp", aoc.No, "0x60")
	res, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
		return
	}
	setMac := false
	if strings.Contains(string(res), "15") {
		setMac = true
	}

	if setMac {
		if err := setScreen(aoc.No, aoc.Mac); err != nil {
			log.Fatal(err)
			return
		}
		if err := setScreen(asus.No, asus.Mac); err != nil {
			log.Fatal(err)
			return
		}
	} else {

		if err := setScreen(aoc.No, aoc.Win); err != nil {
			log.Fatal(err)
			return
		}
		if err := setScreen(asus.No, asus.Win); err != nil {
			log.Fatal(err)
			return
		}
	}
}

func setScreen(screen string, output string) error {
	cmd := exec.Command("./winddcutil.exe", "setvcp", screen, "0x60", output)
	res, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	fmt.Println(res)
	return nil
}
