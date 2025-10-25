package types

type Command struct {
	Depth       int
	DirMode     bool
	Grep        string
	Gsensitive  string
	Host        string
	IgnorePaths []string
	Paths       []string
	ShowHidden  bool
	Vgrep       string
	Vsensitive  string
	StopServer  bool
	EscapeChars string
	Quote       bool
	Trim        string
}

type RunArgs struct {
	Host       string
	Port       int
	Ignore     []string
	IgnoreFile string
	Watch      []string
	WatchFile  string
}

type NetResponse struct {
	Ack   bool // acts as 200 http response when true
	Error string
	Paths string
}
