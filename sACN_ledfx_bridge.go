package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Hundemeier/go-sacn/sacn"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/mattn/go-isatty"
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

var configFromFile bool = false

var configData Config = Config{
	Universe:   1,
	Channel:    1,
	Scenes:     []string{},
	LedFx_host: "http://127.0.0.1:8888",
}
var tempScenes = []string{}

var configFile string

func main() {

	var (
		daemonMode bool
		showHelp   bool
		opts       []tea.ProgramOption
	)

	flag.BoolVar(&daemonMode, "d", false, "run as a daemon")
	flag.StringVar(&configFile, "c", "./config.json", "config file path")
	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.Parse()

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if daemonMode || !isatty.IsTerminal(os.Stdout.Fd()) {
		// If we're in daemon mode don't render the TUI
		opts = []tea.ProgramOption{tea.WithoutRenderer()}
	} else {
		// If we're in TUI mode, discard log output
		log.SetOutput(io.Discard)
	}

	file, err := os.ReadFile(configFile)
	if err == nil {
		err = json.Unmarshal(file, &configData)
		if err != nil {
			log.Fatal(err)
		}
		configFromFile = true
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

	p = tea.NewProgram(initialModel(), opts...)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func loadLedfxScenes() {
	var resp *http.Response
	resp, err := http.Get(
		configData.LedFx_host + "/api/scenes",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var apiObj struct {
		Scenes map[string]struct {
			Name string `json:"name"`
		} `json:"scenes"`
	}

	err = json.Unmarshal(body, &apiObj)
	if err != nil {
		log.Fatal(err)
	}

	tempScenes = tempScenes[:0]
	for k := range apiObj.Scenes {
		tempScenes = append(tempScenes, k)
	}

	slices.Sort(tempScenes)
}

type Styles struct {
	colorText      lipgloss.Color
	colorSelected  lipgloss.Color
	colorError     lipgloss.Color
	colorOK        lipgloss.Color
	colorHighlight lipgloss.Color
	Border         lipgloss.Style
	Header         lipgloss.Style
}

func DefaultStyles() *Styles {
	s := new(Styles)
	s.colorText = lipgloss.Color("7")
	s.colorError = lipgloss.Color("1")
	s.colorSelected = lipgloss.Color("8")
	s.colorOK = lipgloss.Color("2")
	s.colorHighlight = lipgloss.Color("202")
	s.Border = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	s.Header = lipgloss.NewStyle().Foreground(s.colorHighlight)
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
	sceneCursor  int
	settingItems []string
	textInput    textinput.Model
	spinner      spinner.Model
	recieving    bool
	changed      bool
}

var urlRegex = regexp.MustCompile(`(?m)^(?P<protocol>https?):\/\/(?P<host>(?:(?:[a-z0-9\-_]+\b)\.)*\w+\b)(?:\:(?P<port>\d{1,5}))?(?P<path>\/[\/\d\w\.-]*)*(?:\?(?P<query>[^#/]+))?(?:#(?P<fragment>.+))?$`)

func textInputValidatorGen(cursorPos int) textinput.ValidateFunc {
	if cursorPos < 2 {

		var maxVal uint64 = 512
		if cursorPos == 0 {
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

			if urlRegex.FindString(str) != "" {
				return nil
			}
			return errors.New("url invalid")
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
		settingItems: []string{"Universe", "Channel", "LedFx Host", "Scenes", "[Save]"},
		textInput:    ti,
		spinner:      sp,
		recieving:    false,
		changed:      false,
		sceneCursor:  -1,
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
			if !m.textInput.Focused() && m.sceneCursor < 0 {
				return m, tea.Quit
			}

		case "up", "k", "w":
			if m.cursor > 0 && !m.textInput.Focused() && m.sceneCursor < 0 {
				m.cursor--
			} else if m.sceneCursor > 0 {
				m.sceneCursor--
			} else if (m.cursor == 0 || m.cursor == 1) && m.textInput.Focused() {
				i, err := strconv.ParseUint(m.textInput.Value(), 10, 16)
				if err == nil {
					m.textInput.SetValue(fmt.Sprint(i + 1))
				}
			}

		case "down", "j", "s":
			if m.cursor < len(m.settingItems)-1 && !m.textInput.Focused() && m.sceneCursor < 0 {
				m.cursor++
			} else if m.sceneCursor >= 0 && m.sceneCursor < len(tempScenes) {
				m.sceneCursor++
			} else if (m.cursor == 0 || m.cursor == 1) && m.textInput.Focused() {
				i, err := strconv.ParseUint(m.textInput.Value(), 10, 16)
				if err == nil && i > 1 {
					m.textInput.SetValue(fmt.Sprint(i - 1))
				}
			}

		case "enter", " ":
			if (m.cursor == 0 || m.cursor == 1 || m.cursor == 2) && !m.textInput.Focused() {
				// Select Text input
				m.textInput.Validate = textInputValidatorGen(m.cursor)
				m.textInput.Focus()
				m.textInput.SetValue(configValueFromIndex(m.cursor))
			} else if m.cursor == 3 && m.sceneCursor < 0 {
				// Select Scenes menu
				m.sceneCursor = 0
				tempScenes = configData.Scenes[:]
			} else if m.cursor == 3 && m.sceneCursor == 0 {
				loadLedfxScenes()
			} else if msg.String() == "enter" && m.textInput.Focused() && m.textInput.Err == nil {
				// set Text input changes
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
				configFromFile = true
			}
		case "pgup":
			if m.cursor == 3 && m.sceneCursor > 1 {
				tempScenes[m.sceneCursor-2], tempScenes[m.sceneCursor-1] =
					tempScenes[m.sceneCursor-1], tempScenes[m.sceneCursor-2]
				m.sceneCursor--
			}
		case "pgdown":
			if m.cursor == 3 && m.sceneCursor > 0 && m.sceneCursor < len(tempScenes) {
				tempScenes[m.sceneCursor-1], tempScenes[m.sceneCursor-0] =
					tempScenes[m.sceneCursor-0], tempScenes[m.sceneCursor-1]
				m.sceneCursor++
			}
		case "ctrl+s":
			if m.cursor == 3 && m.sceneCursor >= 0 {
				configData.Scenes = tempScenes
				m.sceneCursor = -1
				m.changed = true
			}
		case "esc":
			m.textInput.Blur()
			m.sceneCursor = -1

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
		return fmt.Sprintf("%d Scenes", len(configData.Scenes))
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
				7,
				lipgloss.Center,
				lipgloss.Center,
				title,
			))

	sceneInfo := fmt.Sprintf(" %03d => %s", channelValue, ActiveScene)

	recievingSpinner := m.spinner.View()

	settingsHeader := lipgloss.NewStyle().PaddingLeft(2).Bold(true).
		Foreground(m.styles.colorHighlight).
		Render("Settings:")
	if !configFromFile {
		settingsHeader += " (not saved)"
	} else if m.changed {
		settingsHeader += " (changed)"
	}

	settingsColumn := ""
	valueColumn := ""
	sceneColumn := ""

	for i, setting := range m.settingItems {
		cursor := " " // no cursor
		value := "  " + configValueFromIndex(i)
		lineStyle := colStyle(m.styles.colorText)
		if m.cursor == i {
			lineStyle = colStyle(m.styles.colorSelected)
			if !m.textInput.Focused() {
				if m.sceneCursor < 0 {
					cursor = ">" // cursor!
				}
			} else {
				value = m.textInput.View()
			}
		}

		settingsColumn += lineStyle.Render(fmt.Sprintf("%s %s", cursor, setting)) + "\n"
		valueColumn += value + "\n"
	}

	lineStyle := colStyle(m.styles.colorText)
	if m.sceneCursor == 0 {
		sceneColumn += ">"
		lineStyle = colStyle(m.styles.colorSelected)
	} else {
		sceneColumn += " "
	}
	sceneColumn += " [get scenes from LedFx Api]"
	sceneColumn = lineStyle.Render(sceneColumn) + "\n"

	for i, ts := range tempScenes {
		cursor := " " // no cursor
		lineStyle := colStyle(m.styles.colorText)
		if m.sceneCursor-1 == i {
			cursor = ">" // cursor!
			lineStyle = colStyle(m.styles.colorSelected)
		}
		sceneColumn += lineStyle.Render(fmt.Sprintf("%s %d. %s", cursor, i+1, ts)) + "\n"
	}

	pad := lipgloss.NewStyle().PaddingLeft(2).PaddingRight(2)

	settingsBlock := settingsHeader + "\n\n"
	if m.sceneCursor < 0 {
		settingsBlock += lipgloss.JoinHorizontal(lipgloss.Top, pad.Render(settingsColumn), valueColumn)
	} else {
		settingsBlock += lipgloss.JoinHorizontal(lipgloss.Top, pad.Render(settingsColumn), sceneColumn)
	}

	// The footer
	footer := " Press q to quit."
	if m.textInput.Focused() {
		footer = " Press esc to abort edit or enter to submit."
	}
	if m.sceneCursor >= 0 {
		footer = " Press esc to abort, PgUp or PgDn to reorder scenes or ctrl+S to Save"
	}

	// Send the UI for rendering

	return lipgloss.PlaceVertical(m.height, 0,
		table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(m.styles.Border).
			BorderRow(true).
			Row(header).
			Row(lipgloss.JoinHorizontal(lipgloss.Center, " ", recievingSpinner, "   ", sceneInfo)).
			Row(settingsBlock).
			Row(footer).
			Render())
}
