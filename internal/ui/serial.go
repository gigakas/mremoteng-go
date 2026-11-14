package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
	serialclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/serial"
)

// serialOpener opens a serial port; tests substitute it.
type serialOpener func(serialclient.Config) (streamSession, error)

func defaultSerialOpener(config serialclient.Config) (streamSession, error) {
	conn, err := serialclient.Open(config)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// serialConfig is the line configuration of this panel's connection.
func (p *sessionPanel) serialConfig() serialclient.Config {
	stored := p.profile.Serial
	return serialclient.Config{
		Device:   strings.TrimSpace(p.profile.Host),
		Baud:     stored.Baud,
		DataBits: stored.DataBits,
		Parity:   stored.Parity,
		StopBits: stored.StopBits,
		Flow:     stored.Flow,
	}
}

// connectSerial opens the device and hands it to the terminal view. A serial port has
// no negotiation, so it goes through the same terminal as RAW with a different label.
func (p *sessionPanel) connectSerial() {
	config := p.serialConfig()
	if err := config.Validate(); err != nil {
		p.setProblem("Cannot open the serial port: " + err.Error())
		return
	}
	p.setConnecting("Opening " + config.Description() + " ...")
	go func() {
		session, err := p.screen.openSerial(config)
		fyne.Do(func() {
			if err != nil {
				p.setConnected(false)
				p.setProblem("Serial port failed: " + err.Error())
				return
			}
			p.startStream(session, "serial "+config.LineDescription())
		})
	}()
}

// requestSerialOptions edits the line settings of this connection. They are stored in
// the profile: a device that talks at a different speed is a property of the device,
// not of the window.
func (p *sessionPanel) requestSerialOptions() {
	current := p.profile.Serial
	bauds := serialclient.BaudRates()
	baudLabels := make([]string, 0, len(bauds))
	for _, baud := range bauds {
		baudLabels = append(baudLabels, strconv.Itoa(baud))
	}
	baud := widget.NewSelect(baudLabels, nil)
	baud.SetSelected(strconv.Itoa(orDefault(current.Baud, serialclient.DefaultBaud)))
	bits := widget.NewSelect([]string{"5", "6", "7", "8"}, nil)
	bits.SetSelected(strconv.Itoa(orDefault(current.DataBits, serialclient.DefaultDataBits)))
	parity := widget.NewSelect([]string{"none", "even", "odd", "mark", "space"}, nil)
	parity.SetSelected(orDefaultText(current.Parity, serialclient.DefaultParity))
	stop := widget.NewSelect([]string{"1", "2"}, nil)
	stop.SetSelected(strconv.Itoa(orDefault(current.StopBits, serialclient.DefaultStopBits)))
	flows := serialclient.FlowControls()
	flow := widget.NewSelect(flows, nil)
	flow.SetSelected(orDefaultText(current.Flow, serialclient.DefaultFlow))
	problem := widget.NewLabel("")
	problem.Importance = widget.DangerImportance
	problem.Wrapping = fyne.TextWrapWord

	var popup *modal
	content := container.NewVBox(
		widget.NewLabelWithStyle("Serial line settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Device: "+p.profile.Host),
		widget.NewLabel("Baud rate"), baud,
		widget.NewLabel("Data bits"), bits,
		widget.NewLabel("Parity"), parity,
		widget.NewLabel("Stop bits"), stop,
		widget.NewLabel("Flow control"), flow,
		problem,
		container.NewHBox(
			widget.NewButton("Apply", func() {
				options, err := serialOptionsFromFields(baud.Selected, bits.Selected, parity.Selected, stop.Selected, flow.Selected)
				if err != nil {
					problem.SetText(err.Error())
					return
				}
				if err := p.saveSerialOptions(options); err != nil {
					problem.SetText(err.Error())
					return
				}
				popup.Hide()
				p.setStatus("Serial settings saved: " + p.serialConfig().Description() + ".")
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = p.screen.showModal(content, fyne.NewSize(420, 480))
}

// serialOptionsFromFields turns the four pickers into stored settings, refusing what a
// port cannot use before anything is written.
func serialOptionsFromFields(baud, bits, parity, stop, flow string) (profiles.SerialOptions, error) {
	options := profiles.SerialOptions{Parity: parity, Flow: flow}
	for name, value := range map[string]*int{
		"baud rate": &options.Baud,
	} {
		parsed, err := strconv.Atoi(baud)
		if err != nil {
			return options, fmt.Errorf("the %s %q is not a number", name, baud)
		}
		*value = parsed
	}
	parsedBits, err := strconv.Atoi(bits)
	if err != nil {
		return options, fmt.Errorf("the data bits %q are not a number", bits)
	}
	options.DataBits = parsedBits
	parsedStop, err := strconv.Atoi(stop)
	if err != nil {
		return options, fmt.Errorf("the stop bits %q are not a number", stop)
	}
	options.StopBits = parsedStop
	if err := options.Validate(); err != nil {
		return options, err
	}
	return options, nil
}

// saveSerialOptions stores the line settings in the profile, like the RDP options.
func (p *sessionPanel) saveSerialOptions(options profiles.SerialOptions) error {
	if !p.screen.requireWritable("Saving serial settings") {
		return fmt.Errorf("the profile store is read-only")
	}
	updated, known := p.screen.profileByID(p.profile.ID)
	if !known {
		updated = p.profile
	}
	updated.Serial = options
	if err := updated.Validate(); err != nil {
		return err
	}
	if err := p.screen.ws.putConnection(updated); err != nil {
		return err
	}
	p.profile = updated
	return nil
}

// serialOptionsButton is the session-bar control for the line settings, on serial
// connections only.
func (p *sessionPanel) serialOptionsButton() *hintButton {
	button := newHintButton(nil, "Serial line settings: baud rate, data bits, parity and stop bits", p.requestSerialOptions, p.screen.hint)
	button.SetText("Line")
	button.Importance = widget.LowImportance
	return button
}

// orDefault returns value, or fallback when it is zero.
func orDefault(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

// orDefaultText returns value, or fallback when it is empty.
func orDefaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
