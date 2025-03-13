package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Hundemeier/go-sacn/sacn"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var p *tea.Program

type Config struct {
	Universe   int      `json:"sAcnUniverse"`
	Channel    int      `json:"channel"`
	Scenes     []string `json:"scenes"`
	LedFx_host string   `json:"ledfx_host"`
}

func activateScene(sceneId string, deactivate bool) {
	var action = "activate"
	if deactivate {
		ActiveScene = "OFF"
		action = "deactivate"
	} else {
		ActiveScene = sceneId
	}
	p.Send(updateSceneMsg(ActiveScene))

	payload := map[string]interface{}{"id": sceneId, "action": action}
	out, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut,
		configData.LedFx_host+"/api/scenes",
		strings.NewReader(string(out)),
	)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	_, err = client.Do(req)
	if err != nil {
		// handle error
		log.Fatal(err)
	}

}

var ActiveScene = "OFF"

var channelValue byte = 0
var lastChannelValue byte = 0

var configData Config

func main() {
	file, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("Failed to read file: %v\n", err)
	}

	err = json.Unmarshal(file, &configData)
	if err != nil {
		log.Fatal(err)
	}

	recv, err := sacn.NewReceiverSocket("", nil)
	if err != nil {
		log.Fatal(err)
	}

	recv.SetOnChangeCallback(func(old sacn.DataPacket, newD sacn.DataPacket) {

		channelValue = newD.Data()[configData.Channel-1]

		if channelValue != lastChannelValue {
			lastChannelValue = channelValue

			if channelValue == 0 {
				activateScene(ActiveScene, true)
			} else {
				if channelValue <= byte(len(configData.Scenes)) {
					activateScene(configData.Scenes[channelValue-1], false)
				}
			}
		}
	})
	recv.SetTimeoutCallback(func(univ uint16) {
		//Todo: replace
		// fmt.Println("timeout on", univ)
	})
	recv.Start()

	p = tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type Styles struct {
	Border      lipgloss.Style
	Header      lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.Border = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	s.Header = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	return s
}

type model struct {
	styles *Styles
	width  int
	height int
}

func initialModel() model {
	return model{
		styles: DefaultStyles(),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

type updateSceneMsg string

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case updateSceneMsg:
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	title := `       ___  ______  __  __          ______       ___      _    __        
  ___ / _ |/ ___/ |/ / / /  ___ ___/ / __/_ __  / _ )____(_)__/ /__ ____ 
 (_-</ __ / /__/    / / /__/ -_) _  / _/ \ \ / / _  / __/ / _  / _ '/ -_)
/___/_/ |_\___/_/|_/ /____/\__/\_,_/_/  /_\_\ /____/_/ /_/\_,_/\_, /\__/ 
                                                              /___/      `

	header :=
		m.styles.Header.Render(
			lipgloss.Place(
				m.width-2,
				m.height/4,
				lipgloss.Center,
				lipgloss.Center,
				title,
			))

	sceneInfo := fmt.Sprintf(" %03d => %s", channelValue, ActiveScene)


	// The footer
	quit := "Press q to quit."

	// Send the UI for rendering

	return lipgloss.PlaceVertical(m.height, 0,
		table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(m.styles.Border).
			BorderRow(true).
			Row(header).
			Row(sceneInfo).
			Row().
			Row(quit).
			Render())
}
