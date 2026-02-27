package config

// WireableModule defines a module that can be wired into a project's
// container, config, and server via code injection at marker points.
type WireableModule struct {
	Name        string
	Description string

	// Config injection (pkg/config/config.go)
	ConfigFields string // Struct fields to add
	ConfigLoads  string // Load() assignments to add

	// Container injection (cmd/container.go)
	ContainerImports string // Import lines
	ContainerFields  string // Struct fields
	ModuleInit       string // initModules() code
	BackgroundStart  string // StartBackgroundServices() code
	ContainerHelpers string // Top-level functions/types

	// Server injection (cmd/server.go)
	ServerImports     string // Import lines
	PublicRoutes      string // Public (unauthenticated) routes
	RouteRegistration string // Protected routes
	AuthMiddleware    string // Middleware for protected group

	// External Go dependencies to install
	GoDeps []string

	// Cross-module bridges
	Bridges []Bridge
}

// Bridge defines code to inject when two modules are both wired.
type Bridge struct {
	RequiresModule   string // Other module that must also be wired
	ContainerImports string // Additional imports for bridge
	ContainerInit    string // Code to inject into initModules()
}

// WireableModuleRegistry defines all modules that can be wired into a project.
var WireableModuleRegistry = map[string]WireableModule{
	"jobx": {
		Name:        "jobx",
		Description: "Async job processing with Redis-backed dispatcher",

		ContainerImports: `	"{{GOMODULE}}/pkg/asyncx"`,
		ContainerFields:  `	Dispatcher *asyncx.Dispatcher`,
		ModuleInit:       `	c.Dispatcher = asyncx.NewDispatcher(c.Redis, logx.DefaultLogger())`,
		BackgroundStart:  `	c.Dispatcher.Start(ctx)`,

		Bridges: []Bridge{
			{
				RequiresModule:   "notifx",
				ContainerImports: `	"{{GOMODULE}}/pkg/notifx"`,
				ContainerInit:    `	c.Dispatcher.Register("notifx:send_email", notifx.SendEmailHandler(c.NotificationService))`,
			},
		},
	},

	"notifx": {
		Name:        "notifx",
		Description: "Email notifications via AWS SES",

		ConfigFields: `	Email EmailConfig`,
		ConfigLoads:  `	cfg.Email = loadEmailConfig()`,

		ContainerImports: `	"{{GOMODULE}}/pkg/notifx"
	"{{GOMODULE}}/pkg/notifx/notifxses"`,
		ContainerFields: `	NotificationService notifx.NotificationService`,
		ModuleInit:      `	c.NotificationService = notifxses.NewSESNotifier(c.Config.Email.AWSRegion)`,

		GoDeps: []string{
			"github.com/aws/aws-sdk-go-v2/service/sesv2",
		},

		Bridges: []Bridge{
			{
				RequiresModule:   "jobx",
				ContainerImports: `	"{{GOMODULE}}/pkg/notifx"`,
				ContainerInit:    `	c.Dispatcher.Register("notifx:send_email", notifx.SendEmailHandler(c.NotificationService))`,
			},
		},
	},

	"iam": {
		Name:        "iam",
		Description: "Auth, users, tenants, scopes, API keys",

		ConfigFields: `	JWT           JWTConfig
	Session       SessionConfig
	OTP           OTPConfig
	OAuth         OAuthConfig
	Cookie        CookieConfig
	Invitation    InvitationConfig
	PasswordReset PasswordResetConfig
	APIKey        APIKeyConfig
	Tenant        TenantConfig`,

		ConfigLoads: `	cfg.JWT = loadJWTConfig()
	cfg.Session = loadSessionConfig()
	cfg.OTP = loadOTPConfig()
	cfg.OAuth = loadOAuthConfig()
	cfg.Cookie = loadCookieConfig()
	cfg.Invitation = loadInvitationConfig()
	cfg.PasswordReset = loadPasswordResetConfig()
	cfg.APIKey = loadAPIKeyConfig()
	cfg.Tenant = loadTenantConfig()`,

		ContainerImports: `	"{{GOMODULE}}/pkg/iam/iamcontainer"`,
		ContainerFields:  `	IAM *iamcontainer.Container`,
		ModuleInit: `	c.IAM = iamcontainer.New(iamcontainer.Deps{
		DB:          c.DB,
		Redis:       c.Redis,
		Cfg:         c.Config,
		OTPNotifier: NewConsoleNotifier(),
	})`,
		BackgroundStart: `	c.IAM.StartBackgroundServices(ctx)`,

		ContainerHelpers: `// ConsoleNotifier implements the NotificationService interface
// by printing OTP codes to the terminal/console
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a new console-based OTP notifier
func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

// SendOTP prints the OTP code to the terminal
func (n *ConsoleNotifier) SendOTP(ctx context.Context, contact string, code string) error {
	fmt.Println("\n" + repeatString("=", 60))
	fmt.Println("OTP NOTIFICATION (Console Output)")
	fmt.Println(repeatString("=", 60))
	fmt.Printf("To: %s\n", contact)
	fmt.Printf("Code: %s\n", code)
	fmt.Println(repeatString("=", 60))
	fmt.Println("This is console output for development only")
	fmt.Println("In production, configure email service in config")
	fmt.Println(repeatString("=", 60) + "\n")

	logx.Infof("OTP sent to %s: %s", contact, code)
	return nil
}`,

		PublicRoutes: `	// IAM Routes
	container.IAM.OAuthHandlers.RegisterRoutes(app)
	logx.Info("  > OAuth routes registered")

	container.IAM.PasswordlessHandlers.RegisterRoutes(app)
	logx.Info("  > Passwordless auth routes registered")`,

		AuthMiddleware: `container.IAM.UnifiedAuthMiddleware.Authenticate()`,

		RouteRegistration: `	container.IAM.APIKeyHandlers.RegisterRoutes(protected, container.IAM.UnifiedAuthMiddleware)
	logx.Info("  > API key routes registered")

	container.IAM.InvitationHandlers.RegisterRoutes(protected, container.IAM.UnifiedAuthMiddleware)
	logx.Info("  > Invitation routes registered")`,
	},
}

// IsWireableModule returns true if the given name is a wireable module.
func IsWireableModule(name string) bool {
	_, ok := WireableModuleRegistry[name]
	return ok
}

// WireableModuleNames returns the names of all wireable modules.
func WireableModuleNames() []string {
	var names []string
	for name := range WireableModuleRegistry {
		names = append(names, name)
	}
	return names
}
