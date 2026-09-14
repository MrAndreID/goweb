package config

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/common-nighthawk/go-figure"
)

const MaxNameLength int = 10

func LoadVersion(cfg *Config) {
	var phrase string = "GoWeb"

	if len(cfg.AppName) > MaxNameLength {
		phrase = cfg.AppName[0:MaxNameLength]
	} else {
		phrase = cfg.AppName
	}

	name := figure.NewFigure(phrase, "standard", true)

	name.Print()

	fmt.Println(strings.Repeat("=", widestRow(name.Slicify())) + " " + cfg.AppVersion)

	fmt.Println()
}

func widestRow(rows []string) int {
	var width int

	for _, row := range rows {
		if length := utf8.RuneCountInString(row); length > width {
			width = length
		}
	}

	return width
}
