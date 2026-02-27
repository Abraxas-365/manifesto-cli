package config

// WireableModule defines a module that can be wired into a project's
// container, config, server, and Makefile via code injection at marker points.
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

	// Makefile injection (Makefile)
	MakefileEnv        string // Environment variable blocks (top-level exports)
	MakefileEnvDisplay string // @echo lines for `make env` target (NO leading tab — added by injector)

	// External Go dependencies to install
	GoDeps []string

	// Required source modules (from ModuleRegistry) that must be downloaded
	RequiredModules []string

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
	"fsx": {
		Name:        "fsx",
		Description: "File system abstraction (local, S3)",

		RequiredModules: []string{"fsx"},

		ContainerImports: `	"{{GOMODULE}}/pkg/fsx"
	"{{GOMODULE}}/pkg/fsx/fsxlocal"
	"{{GOMODULE}}/pkg/fsx/fsxs3"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"`,

		ContainerFields: `	FileSystem fsx.FileSystem
	S3Client   *s3.Client`,

		ModuleInit: `	c.initFileStorage()`,

		ContainerHelpers: `func (c *Container) initFileStorage() {
	storageMode := getEnv("STORAGE_MODE", "local")

	switch storageMode {
	case "s3":
		awsRegion := getEnv("AWS_REGION", "us-east-1")
		awsBucket := getEnv("AWS_BUCKET", "{{PROJECTNAME}}-uploads")

		cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion(awsRegion))
		if err != nil {
			logx.Fatalf("Unable to load AWS SDK config: %v", err)
		}
		c.S3Client = s3.NewFromConfig(cfg)
		c.FileSystem = fsxs3.NewS3FileSystem(c.S3Client, awsBucket, "")
		logx.Infof("  S3 file system configured (bucket: %s, region: %s)", awsBucket, awsRegion)

	case "local":
		uploadDir := getEnv("UPLOAD_DIR", "./uploads")
		localFS, err := fsxlocal.NewLocalFileSystem(uploadDir)
		if err != nil {
			logx.Fatalf("Failed to initialize local file system: %v", err)
		}
		c.FileSystem = localFS
		logx.Infof("  Local file system configured (path: %s)", localFS.GetBasePath())

	default:
		logx.Fatalf("Unknown STORAGE_MODE: %s (use 'local' or 's3')", storageMode)
	}
}`,

		MakefileEnv: `# ============================================================================
# Environment Variables - Storage Configuration
# ============================================================================

export STORAGE_MODE = local
export UPLOAD_DIR = ./uploads
export AWS_REGION = us-east-1
export AWS_BUCKET = {{PROJECTNAME}}-uploads`,

		MakefileEnvDisplay: `@echo "Storage:"
@echo "  MODE:              $(STORAGE_MODE)"
@echo "  UPLOAD_DIR:        $(UPLOAD_DIR)"
@echo ""`,

		GoDeps: []string{
			"github.com/aws/aws-sdk-go-v2/config",
			"github.com/aws/aws-sdk-go-v2/service/s3",
		},
	},

	"asyncx": {
		Name:        "asyncx",
		Description: "Async primitives: futures, fan-out, pools, retry, timeout",

		RequiredModules: []string{"asyncx"},
	},

	"ai": {
		Name:        "ai",
		Description: "LLM, embeddings, vector store, OCR, speech",

		RequiredModules: []string{"ai", "fsx"},
	},

	"jobx": {
		Name:        "jobx",
		Description: "Async job processing with Redis-backed dispatcher",

		RequiredModules: []string{"jobx", "asyncx"},

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

		RequiredModules: []string{"notifx"},

		ConfigFields: `	Email EmailConfig`,
		ConfigLoads:  `	cfg.Email = loadEmailConfig()`,

		ContainerImports: `	"{{GOMODULE}}/pkg/notifx"
	"{{GOMODULE}}/pkg/notifx/notifxses"`,
		ContainerFields: `	NotificationService notifx.NotificationService`,
		ModuleInit:      `	c.NotificationService = notifxses.NewSESNotifier(c.Config.Email.AWSRegion)`,

		MakefileEnv: `# ============================================================================
# Environment Variables - Email Configuration
# ============================================================================

export EMAIL_PROVIDER = ses
export EMAIL_FROM_ADDRESS = noreply@{{PROJECTNAME}}.com
export EMAIL_FROM_NAME = {{PROJECTNAME}}

# SMTP Configuration (if using SMTP)
export SMTP_HOST =
export SMTP_PORT = 587
export SMTP_USERNAME =
export SMTP_PASSWORD =

# AWS SES Configuration
export AWS_SES_REGION = us-east-1`,

		MakefileEnvDisplay: `@echo "Email:"
@echo "  PROVIDER:          $(EMAIL_PROVIDER)"
@echo "  FROM:              $(EMAIL_FROM_ADDRESS)"
@echo ""`,

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

		RequiredModules: []string{"iam", "migrations"},

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
	fmt.Println("📧 OTP NOTIFICATION (Console Output)")
	fmt.Println(repeatString("=", 60))
	fmt.Printf("📨 To: %s\n", contact)
	fmt.Printf("🔐 Code: %s\n", code)
	fmt.Println(repeatString("=", 60))
	fmt.Println("⚠️  This is console output for development only")
	fmt.Println("⚠️  In production, configure email service in config")
	fmt.Println(repeatString("=", 60) + "\n")

	logx.Infof("📧 OTP sent to %s: %s", contact, code)
	return nil
}`,

		MakefileEnv: `# ============================================================================
# Environment Variables - JWT Configuration
# ============================================================================

export JWT_SECRET_KEY = development-supersecret-key-must-be-at-least-32-characters-long-change-in-prod
export JWT_ACCESS_TOKEN_TTL = 15m
export JWT_REFRESH_TOKEN_TTL = 168h
export JWT_ISSUER = {{PROJECTNAME}}
export JWT_AUDIENCE = {{PROJECTNAME}}-api,{{PROJECTNAME}}-web

# ============================================================================
# Environment Variables - API Key Configuration
# ============================================================================

export API_KEY_LIVE_PREFIX = {{PROJECTNAME}}_live
export API_KEY_TEST_PREFIX = {{PROJECTNAME}}_test
export API_KEY_TOKEN_LENGTH = 32

# ============================================================================
# Environment Variables - Session Configuration
# ============================================================================

export SESSION_EXPIRATION_TIME = 24h
export SESSION_CLEANUP_INTERVAL = 1h
export SESSION_MAX_PER_USER = 10

# ============================================================================
# Environment Variables - OTP Configuration
# ============================================================================

export OTP_CODE_LENGTH = 6
export OTP_EXPIRATION_TIME = 10m
export OTP_MAX_ATTEMPTS = 5
export OTP_RATE_LIMIT_WINDOW = 1m
export OTP_TOKEN_BYTE_LENGTH = 3

# ============================================================================
# Environment Variables - Invitation Configuration
# ============================================================================

export INVITATION_DEFAULT_EXPIRATION_DAYS = 7
export INVITATION_TOKEN_BYTE_LENGTH = 32
export INVITATION_MAX_PENDING_PER_TENANT = 100

# ============================================================================
# Environment Variables - Password Configuration
# ============================================================================

export PASSWORD_RESET_TOKEN_BYTE_LENGTH = 32
export PASSWORD_RESET_EXPIRATION_TIME = 1h
export PASSWORD_RESET_RATE_LIMIT_WINDOW = 15m
export PASSWORD_RESET_MAX_ATTEMPTS = 3
export BCRYPT_COST = 10

# ============================================================================
# Environment Variables - Cookie Configuration
# ============================================================================

export COOKIE_ACCESS_TOKEN_NAME = access_token
export COOKIE_REFRESH_TOKEN_NAME = refresh_token
export COOKIE_DOMAIN =
export COOKIE_PATH = /
export COOKIE_SECURE = false
export COOKIE_HTTP_ONLY = true
export COOKIE_SAME_SITE = Lax

# ============================================================================
# Environment Variables - OAuth Configuration
# ============================================================================

# Google OAuth
export OAUTH_GOOGLE_ENABLED = false
export OAUTH_GOOGLE_CLIENT_ID =
export OAUTH_GOOGLE_CLIENT_SECRET =
export OAUTH_GOOGLE_REDIRECT_URL = http://localhost:5173/auth/callback/?provider=google
export OAUTH_GOOGLE_SCOPES = openid,email,profile
export OAUTH_GOOGLE_AUTH_URL = https://accounts.google.com/o/oauth2/auth
export OAUTH_GOOGLE_TOKEN_URL = https://oauth2.googleapis.com/token
export OAUTH_GOOGLE_USER_INFO_URL = https://www.googleapis.com/oauth2/v2/userinfo
export OAUTH_GOOGLE_TIMEOUT = 30s

# Microsoft OAuth
export OAUTH_MICROSOFT_ENABLED = false
export OAUTH_MICROSOFT_CLIENT_ID =
export OAUTH_MICROSOFT_CLIENT_SECRET =
export OAUTH_MICROSOFT_REDIRECT_URL = http://localhost:$(SERVER_PORT)/auth/callback/microsoft
export OAUTH_MICROSOFT_SCOPES = openid,email,profile,User.Read
export OAUTH_MICROSOFT_AUTH_URL = https://login.microsoftonline.com/common/oauth2/v2.0/authorize
export OAUTH_MICROSOFT_TOKEN_URL = https://login.microsoftonline.com/common/oauth2/v2.0/token
export OAUTH_MICROSOFT_USER_INFO_URL = https://graph.microsoft.com/v1.0/me
export OAUTH_MICROSOFT_TIMEOUT = 30s

# OAuth State Manager
export OAUTH_STATE_MANAGER_TYPE = redis
export OAUTH_STATE_TTL = 10m

# ============================================================================
# Environment Variables - Tenant Configuration
# ============================================================================

export TENANT_TRIAL_DAYS = 30
export TENANT_SUBSCRIPTION_YEARS = 1
export TENANT_MAX_USERS_BASIC = 5
export TENANT_MAX_USERS_PROFESSIONAL = 50
export TENANT_MAX_USERS_ENTERPRISE = 500`,

		MakefileEnvDisplay: `@echo "JWT:"
@echo "  ISSUER:            $(JWT_ISSUER)"
@echo "  ACCESS_TTL:        $(JWT_ACCESS_TOKEN_TTL)"
@echo "  REFRESH_TTL:       $(JWT_REFRESH_TOKEN_TTL)"
@echo ""
@echo "OAuth:"
@echo "  GOOGLE:            $(OAUTH_GOOGLE_ENABLED)"
@echo "  MICROSOFT:         $(OAUTH_MICROSOFT_ENABLED)"
@echo "  STATE_MANAGER:     $(OAUTH_STATE_MANAGER_TYPE)"
@echo ""`,

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
