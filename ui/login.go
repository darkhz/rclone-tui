package ui

import (
	"github.com/darkhz/rclone-tui/rclone"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// LoginUI stores the authentication parameters.
type LoginUI struct {
	params map[string]string
}

var login LoginUI

// LoginScreen displays a login screen to enter authentication information.
func LoginScreen() {
	var modal *Modal

	setData := func(name string, data interface{}) {
		if login.params == nil {
			login.params = make(map[string]string)
		}

		login.params[name] = data.(string)
	}

	form := NewForm()
	form.SetButtonsAlign(tview.AlignCenter)
	form.AddFormItem(
		GetFormInputField("Host", true, false, setData, func(label string) {}),
	)
	form.AddFormItem(
		GetFormInputField("User", true, false, setData, func(label string) {}),
	)
	form.AddFormItem(
		GetFormInputField("Password", true, true, setData, func(label string) {}),
	)
	form.AddButton("Login", func() {
		if login.params == nil {
			return
		}

		go func(host, user, pass string) {
			StartLoading("Logging in")

			userInfo, err := rclone.Login(host, user, pass)
			if err != nil {
				ErrorMessage("Login", err, struct{}{})
				return
			}

			StopLoading("Logged in")

			App.QueueUpdateDraw(func() {
				modal.Exit()
				MainPage.RemovePage("login")

				SetViewHostname(userInfo)
				InitViewByName("Dashboard")
			})
		}(login.params["Host"], login.params["User"], login.params["Password"])
	})

	SetViewTitle("Login")
	MainPage.AddAndSwitchToPage("login", tview.NewBox().SetBackgroundColor(tcell.ColorDefault), true)

	modal = NewCustomModal("login_form", form, form.GetFormItemCount()+8, 100)
	modal.Show()
}
