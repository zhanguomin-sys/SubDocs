package connect

// Registry values for the myautoapi:// link, written under HKCU so no
// administrator rights are needed.
const (
	ProtocolKey         = `Software\Classes\` + ProtocolScheme
	ProtocolCommandKey  = ProtocolKey + `\shell\open\command`
	ProtocolDescription = "URL:" + ProductName
)

// ProtocolCommand is the shell\open\command value for the assistant at exe.
func ProtocolCommand(exe string) string {
	return `"` + exe + `" "%1"`
}
