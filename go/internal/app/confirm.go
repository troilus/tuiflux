package app

import (
	"github.com/rivo/tview"
)

func showConfirm(app *App, msg string, fn func(bool)) {
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.pages.RemovePage("confirm")
			fn(buttonLabel == "Yes")
		})
	app.pages.AddAndSwitchToPage("confirm", modal, true)
}
