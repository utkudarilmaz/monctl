package tray

import (
	"fmt"
	"image/color"

	"fyne.io/systray"
)

const maxInputs = 8

// Run shows the tray menu and blocks until the user quits. It must be called
// from the main goroutine.
func Run(ctl Controller) error {
	if err := checkHost(); err != nil {
		return err
	}
	systray.Run(func() { build(ctl) }, nil)
	return nil
}

func build(ctl Controller) {
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

	actions := make(chan func(), 1)
	on := func(item *systray.MenuItem, action func()) {
		go func() {
			for range item.ClickedCh {
				actions <- action
			}
		}()
	}

	var shown []InputView
	refresh := func() {
		view := Render(ctl.State())
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

	on(brighter, func() { _ = ctl.AdjustBrightness(Step) })
	on(dimmer, func() { _ = ctl.AdjustBrightness(-Step) })
	for _, percent := range Presets {
		on(presets.AddSubMenuItem(fmt.Sprintf("%d%%", percent), ""), func() { _ = ctl.SetBrightness(percent) })
	}
	for i, item := range inputs {
		on(item, func() {
			if i < len(shown) {
				_ = ctl.SetInput(shown[i].ID)
			}
		})
	}
	on(quit, systray.Quit)

	go func() {
		refresh()
		for {
			select {
			case action := <-actions:
				action()
				refresh()
			case <-systray.TrayOpenedCh:
				refresh()
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
