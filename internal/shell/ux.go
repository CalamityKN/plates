package shell

import (
	"math/rand/v2"
)

const Version = "0.8.0"

const banner = `██████╗ ██╗      █████╗ ████████╗███████╗███████╗
██╔══██╗██║     ██╔══██╗╚══██╔══╝██╔════╝██╔════╝
██████╔╝██║     ███████║   ██║   █████╗  ███████╗
██╔═══╝ ██║     ██╔══██║   ██║   ██╔══╝  ╚════██║
██║     ███████╗██║  ██║   ██║   ███████╗███████║
╚═╝     ╚══════╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚══════╝

      Plate-Based Command Rendering System
`

var tips = []string{
	"Tip: Use 'show ingredients' after loading a plate to see missing values.",
	"Tip: Use 'setg my_ip <ip>' to store your local IP in the pantry.",
	"Tip: Use 'lint rack' after adding new plates.",
	"Tip: Use 'save output' to keep the last rendered command.",
	"Tip: Use 'search plates <query>' to find plates by tags and ingredients.",
}

var fortunes = []string{
	"A clean rack is a fast rack.",
	"Measure twice, stamp once.",
	"The best command is the one you understand before you paste it.",
	"Render with intent; copy with care.",
	"Good templates turn repetition into memory.",
}

func randomTip() string {
	return tips[rand.IntN(len(tips))]
}

func randomFortune() string {
	return fortunes[rand.IntN(len(fortunes))]
}

func randomIndex(length int) int {
	return rand.IntN(length)
}
