package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/anomalyco/tuiflux/internal/api"
	"github.com/anomalyco/tuiflux/internal/config"
	"github.com/anomalyco/tuiflux/internal/locale"
)

func showSettings(app *App) {
	cfg := app.cfg

	form := tview.NewForm()
	form.SetBorder(true).SetTitle(app.L["settings"])

	serverURL := cfg.ServerURL
	apiKey := cfg.APIKey
	lang := cfg.Language
	verifySSL := cfg.VerifySSL

	form.AddInputField(app.L["settings_server_url"], serverURL, 0, nil, func(text string) {
		serverURL = text
	})
	form.AddInputField(app.L["settings_api_key"], apiKey, 0, nil, func(text string) {
		apiKey = text
	})
	form.AddDropDown(app.L["settings_language"], []string{"en", "cn"}, 0, func(option string, index int) {
		if index >= 0 {
			lang = option
		}
	})
	form.AddDropDown(app.L["settings_verify_ssl"], []string{"Yes", "No"}, 0, func(option string, index int) {
		if index >= 0 {
			verifySSL = option == "Yes"
		}
	})

	langIdx := 0
	if cfg.Language == "cn" {
		langIdx = 1
	}
	sslIdx := 0
	if !cfg.VerifySSL {
		sslIdx = 1
	}
	if dd, ok := form.GetFormItem(2).(*tview.DropDown); ok {
		dd.SetCurrentOption(langIdx)
	}
	if dd, ok := form.GetFormItem(3).(*tview.DropDown); ok {
		dd.SetCurrentOption(sslIdx)
	}

	form.AddButton(app.L["settings_save"], func() {
		app.cfg.ServerURL = serverURL
		app.cfg.APIKey = apiKey
		app.cfg.Language = lang
		app.cfg.VerifySSL = verifySSL

		config.Save(app.cfg)

		app.L = locale.Get(lang)
		app.api = api.New(serverURL, apiKey, verifySSL)
		app.currentFeedID = 0
		app.entries = nil
		app.exhaustedFeeds = make(map[int]bool)

		app.pages.RemovePage("settings")
		app.pages.SwitchToPage("main")
		go app.initialLoad()
	})

	form.AddButton(app.L["settings_cancel"], func() {
		app.pages.RemovePage("settings")
		app.pages.SwitchToPage("main")
	})

	form.SetInputCapture(defaultEscapeCapture(app))

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(form, 0, 3, true).
			AddItem(nil, 0, 1, false),
			0, 3, true).
		AddItem(nil, 0, 1, false)

	app.pages.AddAndSwitchToPage("settings", flex, true)
}

func defaultEscapeCapture(a *App) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.pages.SwitchToPage("main")
			return nil
		}
		return event
	}
}
