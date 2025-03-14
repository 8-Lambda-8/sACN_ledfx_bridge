package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Hundemeier/go-sacn/sacn"
	"github.com/charmbracelet/bubbles/textinput"
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
	Border lipgloss.Style
	Header lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.Border = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	s.Header = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	return s
}

type model struct {
	styles       *Styles
	width        int
	height       int
	cursor       int
	settingItems []string
	textInput    textinput.Model

	changed bool
}

func initialModel() model {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.Width = 50
	//Todo: add Validator

	return model{
		styles:       DefaultStyles(),
		settingItems: []string{"Universe", "Channel", "LedFx Host", "Scenes", "Save"},
		changed:      false,
		textInput:    ti,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return textinput.Blink
}

type updateSceneMsg string
type errMsg error

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !m.textInput.Focused() {
				return m, tea.Quit
			}

		case "up", "k", "w":
			if m.cursor > 0 && !m.textInput.Focused() {
				m.cursor--
			}

		case "down", "j", "s":
			if m.cursor < len(m.settingItems)-1 && !m.textInput.Focused() {
				m.cursor++
			}

		case "enter", " ":
			if (m.cursor == 0 || m.cursor == 1 || m.cursor == 2) && !m.textInput.Focused() {
				m.textInput.Focus()
				m.textInput.SetValue(configValueFromIndex(m.cursor))
			} else if msg.String() == "enter" && m.textInput.Focused() && m.textInput.Err == nil {
				// set changes
				value := m.textInput.Value()
				switch m.cursor {
				case 0:
					i, err := strconv.Atoi(value)
					if err == nil {
						if configData.Universe != i {
							m.changed = true
						}
						configData.Universe = i
					}
				case 1:
					i, err := strconv.Atoi(value)
					if err == nil {
						if configData.Channel != i {
							m.changed = true
						}
						configData.Channel = i
					}
				case 2:
					if configData.LedFx_host != value {
						m.changed = true
					}
					configData.LedFx_host = value
				}

				m.textInput.Blur()
			}

		case "esc":
			m.textInput.Blur()

		case "tab":
			m.textInput.Reset()
		}
	case updateSceneMsg:

	case errMsg:
		// ToDo: Handle this
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func configValueFromIndex(index int) string {
	switch index {
	case 0:
		return fmt.Sprintf("%d", configData.Universe)
	case 1:
		return fmt.Sprintf("%d", configData.Channel)
	case 2:
		return configData.LedFx_host
	case 3:
		return "(not working yet, set scene ids in config.json)"
	default:
		return ""
	}
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

	settings := " Settings:"
	if m.changed {
		settings += " (changed)"
	}

	settingsTable := table.New().
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderBottom(false).
		BorderColumn(true)

	for i, setting := range m.settingItems {
		cursor := " " // no cursor
		value := "  " + configValueFromIndex(i)
		if m.cursor == i {
			if !m.textInput.Focused() {
				cursor = ">" // cursor!
			} else {
				value = m.textInput.View()
			}
		}

		settingsTable.Row(
			fmt.Sprintf("  %s %s  ", cursor, setting),
			value,
		)
	}

	// The footer
	quit := " Press q to quit."

	// Send the UI for rendering

	return lipgloss.PlaceVertical(m.height, 0,
		table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(m.styles.Border).
			BorderRow(true).
			Row(header).
			Row(sceneInfo).
			Row(settings+"\n"+settingsTable.Render()).
			Row(quit).
			Render())
}
