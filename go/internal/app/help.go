package app

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func showHelp(app *App) {
	text := fmt.Sprintf(`[::b]Tuiflux[::]

%s: %s

  q  — %s
  f  — %s
  Tab — %s
  Enter — %s
  m  — %s
  Space — %s
  s  — %s
  o  — %s
  r  — %s
  Insert — %s
  Delete — %s
  PageUp — %s
  PageDown — %s`,
		app.L["help_version"], Version,
		app.L["quit"],
		app.L["refresh"],
		app.L["switch_pane"],
		app.L["read_more"],
		app.L["toggle_read"],
		app.L["read_and_next"],
		app.L["star_unstar"],
		app.L["open_in_browser"],
		app.L["mark_page_read"],
		app.L["prev_feed"],
		app.L["next_feed"],
		app.L["page_up"],
		app.L["page_down"],
	)

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetText(text).
		SetTextAlign(tview.AlignLeft)
	tv.SetBorder(true).SetTitle(app.L["help"])

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tv, 0, 3, true).
			AddItem(nil, 0, 1, false),
			0, 3, true).
		AddItem(nil, 0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.pages.SwitchToPage("main")
			return nil
		}
		return event
	})

	app.pages.AddAndSwitchToPage("help", flex, true)
}
