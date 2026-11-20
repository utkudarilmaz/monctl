package tray

import (
	"fmt"
	"image/color"

	"fyne.io/systray"
)

const maxInputs = 8

// Run shows the tray menu and blocks until the user quits. It must be called
// from the main goroutine. Errors from the monitor go to report.
func Run(ctl Controller, report func(error)) error {
	if err := checkHost(); err != nil {
		return err
	}
	systray.Run(func() { build(ctl, report) }, nil)
	return nil
}

func build(ctl Controller, report func(error)) {
	systray.SetTemplateIcon(Icon(color.Black), Icon(color.White))
	systray.SetTooltip("monctl")

	status := systray.AddMenuItem("monctl", "")
	status.Disable()
	systray.AddSeparator()
	brighter := systray.AddMenuItem(fmt.Sprintf("Brighter (+%d)", Step), "")
	dimmer := systray.AddMenuItem(fmt.Sprintf("Dimmer (-%d)", Step), "")
	presets := systray.AddMenuItem("Set brightness", "")
	systray.AddSeparator()
	header := systray.AddMenuItem("Send picture to", "")
	header.Disable()
	var inputs [maxInputs]*systray.MenuItem
	for i := range inputs {
		inputs[i] = systray.AddMenuItem("", "")
		inputs[i].Hide()
	}
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "")

	actions := make(chan func() error, 1)
	on := func(item *systray.MenuItem, action func() error) {
		go func() {
			for range item.ClickedCh {
				actions <- action
			}
		}()
	}

	var shown []InputView
	reportState := once(report)
	refresh := func(failure error) {
		state, err := ctl.State()
		reportState(err)
		view := Render(state, err)
		if failure != nil {
			report(failure)
			view = view.Failed()
		}
		shown = view.Inputs
		status.SetTitle(view.Status)
		for _, item := range []*systray.MenuItem{brighter, dimmer, presets} {
			setEnabled(item, view.Enabled)
		}
		if len(view.Inputs) == 0 {
			header.Hide()
		} else {
			header.Show()
		}
		for i, item := range inputs {
			if i >= len(view.Inputs) {
				item.Hide()
				continue
			}
			in := view.Inputs[i]
			item.SetTitle(in.Title)
			setEnabled(item, !in.Current)
			if in.Current {
				item.Check()
			} else {
				item.Uncheck()
			}
			item.Show()
		}
	}

	on(brighter, func() error { return ctl.AdjustBrightness(Step) })
	on(dimmer, func() error { return ctl.AdjustBrightness(-Step) })
	for _, percent := range Presets {
		on(presets.AddSubMenuItem(fmt.Sprintf("%d%%", percent), ""), func() error { return ctl.SetBrightness(percent) })
	}
	for i, item := range inputs {
		on(item, func() error {
			if i >= len(shown) {
				return nil
			}
			return ctl.SetInput(shown[i].ID)
		})
	}
	go func() {
		<-quit.ClickedCh
		systray.Quit()
	}()

	go func() {
		refresh(nil)
		for {
			select {
			case action := <-actions:
				refresh(action())
			case <-systray.TrayOpenedCh:
				refresh(nil)
			}
		}
	}()
}

func setEnabled(item *systray.MenuItem, enabled bool) {
	if enabled {
		item.Enable()
	} else {
		item.Disable()
	}
}
