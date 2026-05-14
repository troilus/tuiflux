package app

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/anomalyco/tuiflux/internal/models"
)

func showReader(app *App, entry *models.Entry) {
	starStatus := app.L["unstarred"]
	if entry.Starred {
		starStatus = app.L["starred"]
	}

	statusText := fmt.Sprintf("%s: %s | %s", app.L["status"], entry.Status, starStatus)

	content := fmt.Sprintf("# %s\n\n[%s: %s]\n\n%s",
		entry.Title, app.L["source"], entry.URL, htmlToMarkdown(entry.Content))

	header := tview.NewTextView().
		SetText(statusText).
		SetTextAlign(tview.AlignRight).
		SetDynamicColors(true).
		SetTextColor(tcell.ColorWhite)

	body := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetText(content).
		SetScrollable(true)

	body.SetBorder(true).SetTitle(entry.Title)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(body, 0, 1, true)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'm':
			go app.toggleReadEntry(entry)
			return nil
		case 's':
			go app.toggleStarEntry(entry)
			return nil
		case 'o':
			openURL(entry.URL)
			return nil
		}
		switch event.Key() {
		case tcell.KeyEscape:
			app.pages.SwitchToPage("main")
			return nil
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn:
			return event
		}
		return event
	})

	app.pages.AddAndSwitchToPage("reader", flex, true)
}
