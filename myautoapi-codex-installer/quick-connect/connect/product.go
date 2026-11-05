// Package connect holds everything the quick-connect assistant does besides
// drawing its window: link parsing, key checks and the Codex files it writes.
// It has no GUI dependency, so the whole flow is tested on Linux.
package connect

const (
	ProductName = "MyAutoAPI中转快速接入助手"
	Version     = "1.1.4"

	// BaseURL is the only address the assistant ever writes or calls. Import
	// links never carry an address.
	BaseURL      = "https://api.myautoapi.com"
	OfficialSite = "https://myautoapi.com"
	CodexModel   = "gpt-6.1-sol"

	QQGroupNumber = "685861462"
	QQGroupName   = "GPT 与 API 交流 3 群"
	QQGroupURL    = "https://docs.myautoapi.com/contact"

	ProtocolScheme = "myautoapi"

	// Official download pages offered when the app is not installed.
	ChatGPTDownloadURL = "https://chatgpt.com/download/"
	ClaudeDownloadURL  = "https://claude.com/download"
)
