//go:build windows

package connect

import "golang.org/x/sys/windows/registry"

// RegisterProtocol points myautoapi:// at exe. It runs on every start, so a
// moved exe is fixed by opening it once.
func RegisterProtocol(exe string) error {
	root, _, err := registry.CreateKey(registry.CURRENT_USER, ProtocolKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.SetStringValue("", ProtocolDescription); err != nil {
		return err
	}
	if err := root.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	command, _, err := registry.CreateKey(registry.CURRENT_USER, ProtocolCommandKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer command.Close()
	return command.SetStringValue("", ProtocolCommand(exe))
}
