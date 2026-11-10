package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/vault"
)

// requestVault opens the master-password dialog: create a new vault or unlock an
// existing one, and lock it again.
func (s *screen) requestVault() {
	if s.vaultPath == "" {
		path, err := vault.DefaultPath()
		if err != nil {
			s.setProblem("Could not locate the credential file: " + err.Error())
			return
		}
		s.vaultPath = path
	}
	password := widget.NewEntry()
	password.Password = true
	password.SetPlaceHolder("Master password (min 8 characters)")
	state := "Locked"
	if !s.vault.Locked() {
		state = "Unlocked"
	}
	notice := widget.NewLabel("Credential vault: " + state)
	notice.Importance = widget.LowImportance
	var popup *widget.PopUp
	content := container.NewVBox(
		widget.NewLabelWithStyle("Credentials", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		notice,
		password,
		container.NewHBox(
			widget.NewButton("Unlock", func() {
				popup.Hide()
				s.unlockVault(password.Text)
			}),
			widget.NewButton("Create", func() {
				popup.Hide()
				s.createVault(password.Text)
			}),
			widget.NewButton("Lock", func() {
				popup.Hide()
				s.lockVault()
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	)
	popup = widget.NewModalPopUp(container.NewGridWrap(fyne.NewSize(560, 220), container.NewPadded(content)), s.win.Canvas())
	popup.Show()
}

// createVault initialises a new credential file.
func (s *screen) createVault(password string) {
	created, err := vault.Create(s.vaultPath, password)
	if err != nil {
		s.setProblem("Could not create the vault: " + err.Error())
		return
	}
	s.vault = created
	if err := s.vault.Save(); err != nil {
		s.setProblem("Could not save the vault: " + err.Error())
		return
	}
	s.setStatus("Credential vault created and unlocked. Passwords are stored encrypted with your master password.")
}

// unlockVault opens an existing credential file.
func (s *screen) unlockVault(password string) {
	opened, err := vault.Open(s.vaultPath, password)
	if err != nil {
		s.setProblem("Could not unlock the vault: " + err.Error())
		return
	}
	s.vault = opened
	s.setStatus("Credential vault unlocked.")
}

// lockVault forgets the key.
func (s *screen) lockVault() {
	if s.vault != nil {
		s.vault.Lock()
	}
	s.vault = nil
	s.setStatus("Credential vault locked.")
}

// passwordFor returns the typed password, or the stored secret for the profile.
func (p *sessionPanel) passwordFor() string {
	if typed := p.password.Text; typed != "" {
		return typed
	}
	if secret, ok := p.screen.vault.Get(p.profile.ID); ok {
		return secret
	}
	return ""
}

// requestStorePassword saves the typed password in the unlocked vault.
func (p *sessionPanel) requestStorePassword() {
	if p.screen.vault.Locked() {
		p.setProblem("Unlock the credential vault first (File > Credentials).")
		return
	}
	if p.password.Text == "" {
		p.setProblem("Type the password before storing it.")
		return
	}
	if err := p.screen.vault.Set(p.profile.ID, p.password.Text); err != nil {
		p.setProblem("Could not store the password: " + err.Error())
		return
	}
	if err := p.screen.vault.Save(); err != nil {
		p.setProblem("Could not save the vault: " + err.Error())
		return
	}
	p.setStatus("Password stored for " + p.profile.Name + " in the encrypted vault.")
}

// vaultButton is the "store password" action shown on session bars.
func (p *sessionPanel) vaultButton() *widget.Button {
	button := widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), p.requestStorePassword)
	button.Importance = widget.LowImportance
	return button
}
