package config

// Action represents the core functionality of OliveTin - commands that show up
// as buttons in the UI.
type Action struct {
	ID            string
	Title         string
	Icon          string
	Shell         string
	CSS           map[string]string `mapstructure:"omitempty"`
	Timeout       int
	Acls          []string
	ExecOnStartup bool
	ExecOnCron    []string
	MaxConcurrent int
	Arguments     []ActionArgument
	PopupOnStart  bool
}

// ActionArgument objects appear on Actions.
type ActionArgument struct {
	Name        string
	Title       string
	Description string
	Type        string
	Default     string
	Choices     []ActionArgumentChoice
}

// ActionArgumentChoice represents a predefined choice for an argument.
type ActionArgumentChoice struct {
	Value string
	Title string
}

// Entity represents a "thing" that can have multiple actions associated with it.
// for example, a media player with a start and stop action.
type Entity struct {
	Title   string
	Icon    string
	Actions []Action `mapstructure:"actions"`
	CSS     map[string]string
}

// PermissionsList defines what users can do with an action.
type PermissionsList struct {
	View bool
	Exec bool
}

// AccessControlList defines what permissions apply to a user or user group.
type AccessControlList struct {
	Name             string
	AddToEveryAction bool
	MatchUsergroups  []string
	MatchUsernames   []string
	Permissions      PermissionsList
}

type ExternalProxy struct {
	BaseURL string
	Target 	string
}

// Config is the global config used through the whole app.
type Config struct {
	UseSingleHTTPFrontend           bool
	ThemeName                       string
	ListenAddressSingleHTTPFrontend string
	ListenAddressWebUI              string
	ListenAddressRestActions        string
	ListenAddressGrpcActions        string
	ExternalRestAddress             string
	ProxyBaseURL					string
	AuthUser						string
	AuthPass						string
	ExternalProxies					[]ExternalProxy `mapstructure:"externalproxies"`
	LogLevel                        string
	Actions                         []Action `mapstructure:"actions"`
	Entities                        []Entity `mapstructure:"entities"`
	CheckForUpdates                 bool
	PageTitle                       string
	ShowFooter                      bool
	ShowNavigation                  bool
	ShowNewVersions                 bool
	AuthJwtCookieName               string
	AuthJwtSecret                   string // mutually exclusive with pub key config fields
	AuthJwtClaimUsername            string
	AuthJwtClaimUserGroup           string
	AuthJwtPubKeyPath               string // will read pub key from file on disk
	AuthHttpHeaderUsername          string
	AuthHttpHeaderUserGroup         string
	DefaultPermissions              PermissionsList
	AccessControlLists              []AccessControlList
	WebUIDir                        string
}

// DefaultConfig gets a new Config structure with sensible default values.
func DefaultConfig() *Config {
	config := Config{}
	config.UseSingleHTTPFrontend = true
	config.PageTitle = "Control Center"
	config.ShowFooter = false
	config.ShowNavigation = true
	config.ShowNewVersions = true
	config.ListenAddressSingleHTTPFrontend = "0.0.0.0:1337"
	config.ListenAddressRestActions = "localhost:1338"
	config.ListenAddressGrpcActions = "localhost:1339"
	config.ListenAddressWebUI = "localhost:1340"
	config.ExternalRestAddress = "."
	config.ProxyBaseURL = "/"
	AuthUser = ""
	AuthPass = ""
	config.LogLevel = "INFO"
	config.CheckForUpdates = false
	config.DefaultPermissions.Exec = true
	config.DefaultPermissions.View = true
	config.AuthJwtClaimUsername = "name"
	config.AuthJwtClaimUserGroup = "group"
	config.WebUIDir = "./webui"

	return &config
}
