package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Hundemeier/go-sacn/sacn"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var p *tea.Program

type Config struct {
	Universe   uint64   `json:"sAcnUniverse"`
	Channel    uint64   `json:"channel"`
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

var configFile = "./config.json"

func main() {
	file, err := os.ReadFile(configFile)
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
		if newD.Universe() == uint16(configData.Universe) {
			p.Send(recievingMsg(newD.Universe()))
		}

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
		if univ == uint16(configData.Universe) {
			p.Send(timeOutMsg(univ))
		}
	})
	recv.Start()

	p = tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type Styles struct {
	colorText     lipgloss.Color
	colorSelected lipgloss.Color
	colorError    lipgloss.Color
	colorOK       lipgloss.Color
	Border        lipgloss.Style
	Header        lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.colorText = lipgloss.Color("7")
	s.colorError = lipgloss.Color("1")
	s.colorSelected = lipgloss.Color("8")
	s.colorOK = lipgloss.Color("2")
	s.Border = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	s.Header = lipgloss.NewStyle().Foreground(lipgloss.Color("202"))
	return s
}

func colStyle(col lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(col)
}

type model struct {
	styles       *Styles
	width        int
	height       int
	cursor       int
	settingItems []string
	textInput    textinput.Model
	spinner      spinner.Model
	recieving    bool
	changed      bool
}

func textInputValidatorGen(cursorPos int) textinput.ValidateFunc {
	if cursorPos < 2 {

		var maxVal uint64 = 512
		if cursorPos == 1 {
			maxVal = 65279
		}
		return textinput.ValidateFunc(func(str string) error {
			i, err := strconv.ParseUint(str, 10, 16)
			if err != nil {
				return err
			}
			if 1 > i || i > uint64(maxVal) {
				return fmt.Errorf("input out of Range")
			}
			return nil
		})
	} else {
		return textinput.ValidateFunc(func(str string) error {
			// Todo Validate URL
			return nil
		})
	}
}

func initialModel() model {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.Width = 50

	sp := spinner.New()
	sp.Spinner = spinner.Points
	sp.Spinner.FPS = time.Second / 4

	return model{
		styles:       DefaultStyles(),
		settingItems: []string{"Universe", "Channel", "LedFx Host", "Scenes", "Save"},
		textInput:    ti,
		spinner:      sp,
		recieving:    false,
		changed:      false,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
	)
}

type updateSceneMsg string
type recievingMsg uint
type timeOutMsg uint
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
				m.textInput.Validate = textInputValidatorGen(m.cursor)
				m.textInput.Focus()
				m.textInput.SetValue(configValueFromIndex(m.cursor))
			} else if msg.String() == "enter" && m.textInput.Focused() && m.textInput.Err == nil {

				// set changes
				value := m.textInput.Value()
				switch m.cursor {
				case 0:
					i, err := strconv.ParseUint(value, 10, 16)
					if err == nil {
						if configData.Universe != i {
							m.changed = true
						}
						configData.Universe = i
					}
				case 1:
					i, err := strconv.ParseUint(value, 10, 16)
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
			} else if m.cursor == 4 && m.changed {
				//Save config.json
				out, err := json.MarshalIndent(configData, "", "  ")
				if err != nil {
					log.Fatal(err)
				}

				err = os.WriteFile(configFile, out, fs.ModeDevice)
				if err != nil {
					log.Fatalf("Failed to write file: %v\n", err)
				}

				m.changed = false
			}

		case "esc":
			m.textInput.Blur()

		case "tab":
			m.textInput.Reset()
		}

	case updateSceneMsg:

	case spinner.TickMsg:
		var cmd tea.Cmd
		if m.recieving {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	case recievingMsg:
		m.recieving = true
		m.spinner.Style = colStyle(m.styles.colorOK)

		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(m.spinner.Tick())

		return m, cmd

	case timeOutMsg:
		m.recieving = false
		m.spinner.Style = colStyle(m.styles.colorError)

	case errMsg:
		// ToDo: Handle this
		return m, nil
	}

	if m.textInput.Err != nil {
		m.textInput.TextStyle = colStyle(m.styles.colorError)
	} else {
		m.textInput.TextStyle = colStyle(m.styles.colorSelected)
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

	recievingSpinner := m.spinner.View()

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
		lineStyle := colStyle(m.styles.colorText)
		if m.cursor == i {
			lineStyle = colStyle(m.styles.colorSelected)
			if !m.textInput.Focused() {
				cursor = ">" // cursor!
			} else {
				value = m.textInput.View()
			}
		}

		settingsTable.Row(
			lineStyle.Render(fmt.Sprintf("  %s %s  ", cursor, setting)),
			value,
		)
	}

	// The footer
	footer := " Press q to quit."
	if m.textInput.Focused() {
		footer = " Press esc to abort edit or enter to submit."
	}

	// Send the UI for rendering

	return lipgloss.PlaceVertical(m.height, 0,
		table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(m.styles.Border).
			BorderRow(true).
			Row(header).
			Row(lipgloss.JoinHorizontal(lipgloss.Center, " ", recievingSpinner, "   ", sceneInfo)).
			Row(settings+"\n"+settingsTable.Render()).
			Row(footer).
			Render())
}
