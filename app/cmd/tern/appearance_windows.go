package main

import (
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const personalize = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`

// systemDark reports whether Windows apps are in dark mode.
func systemDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, personalize, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	return err == nil && v == 0
}

// watchDark returns whether the system is in dark mode and calls changed
// whenever that changes. A goroutine sleeps on the registry key's change
// notification, so it costs nothing while nothing changes.
func watchDark(changed func(dark bool)) bool {
	dark := systemDark()
	go func() {
		k, err := registry.OpenKey(registry.CURRENT_USER, personalize, registry.NOTIFY)
		if err != nil {
			return
		}
		defer k.Close()
		last := dark
		for {
			if windows.RegNotifyChangeKeyValue(windows.Handle(k), false, windows.REG_NOTIFY_CHANGE_LAST_SET, 0, false) != nil {
				return
			}
			if d := systemDark(); d != last {
				last = d
				changed(d)
			}
		}
	}()
	return dark
}
