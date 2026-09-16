package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ThreeXUIProvider{}

type ThreeXUIProvider struct {
	version string
}

type ThreeXUIProviderModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	BasePath           types.String `tfsdk:"base_path"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	BootstrapUsername  types.String `tfsdk:"bootstrap_username"`
	BootstrapPassword  types.String `tfsdk:"bootstrap_password"`
	TwoFactorCode      types.String `tfsdk:"two_factor_code"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	RequestTimeout     types.String `tfsdk:"request_timeout"`
	MaxRetries         types.Int64  `tfsdk:"max_retries"`
}

const (
	// defaultMaxRetries is the default number of additional attempts on
	// transient 5xx for idempotent write endpoints when max_retries is
	// unset.
	defaultMaxRetries = 1
	// maxAllowedRetries caps user-supplied max_retries to prevent
	// pathological configurations (e.g. 1000 retries × 500ms = 500s spent
	// on a single failing request). Real upstream contention spikes
	// resolve well within a single retry; values above this limit hide
	// real bugs rather than absorb flakes.
	maxAllowedRetries = 10
)

const (
	envEndpoint           = "THREEXUI_ENDPOINT"
	envBasePath           = "THREEXUI_BASE_PATH"
	envUsername           = "THREEXUI_USERNAME"
	envPassword           = "THREEXUI_PASSWORD"
	envInsecureSkipVerify = "THREEXUI_INSECURE_SKIP_VERIFY"
	envRequestTimeout     = "THREEXUI_REQUEST_TIMEOUT"
	envMaxRetries         = "THREEXUI_MAX_RETRIES"
)

// envString returns the first non-empty string: HCL value (if set), then
// THREEXUI_* environment variable, then fallback.
func envString(tfVal types.String, envKey, fallback string) string {
	if !tfVal.IsNull() && !tfVal.IsUnknown() {
		return tfVal.ValueString()
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ThreeXUIProvider{version: version}
	}
}

func (p *ThreeXUIProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "threexui"
	resp.Version = p.version
}

func (p *ThreeXUIProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the 3x-ui panel, e.g. http://localhost:2053. Can also be set via THREEXUI_ENDPOINT environment variable.",
				Validators:  endpointValidators(),
			},
			"base_path": schema.StringAttribute{
				Optional:    true,
				Description: "Base path configured in 3x-ui (webBasePath). Default is '/'. Can also be set via THREEXUI_BASE_PATH environment variable.",
				Validators:  basePathValidators(),
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "3x-ui username. Default is admin. Can also be set via THREEXUI_USERNAME environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "3x-ui password. Default is admin. Can also be set via THREEXUI_PASSWORD environment variable.",
			},
			"bootstrap_username": schema.StringAttribute{
				Optional:    true,
				Description: "Bootstrap username for explicit first-run credential rotation. On 3x-ui v2.9.x it is tried before the primary username/password to avoid exposing the desired password in failed-login logs; on 3x-ui v3.x it is tried only after the primary credentials are rejected. Must be set together with bootstrap_password.",
			},
			"bootstrap_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bootstrap password for explicit first-run credential rotation. On 3x-ui v2.9.x it is tried before the primary username/password to avoid exposing the desired password in failed-login logs; on 3x-ui v3.x it is tried only after the primary credentials are rejected. Must be set together with bootstrap_username.",
			},
			"two_factor_code": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "TOTP code for 2FA login. Sent with the initial authentication request; automatic re-login will fail once the code expires.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification (useful for self-signed certs). Can also be set via THREEXUI_INSECURE_SKIP_VERIFY environment variable.",
			},
			"request_timeout": schema.StringAttribute{
				Optional:    true,
				Description: "HTTP request timeout (e.g. 30s, 1m). Can also be set via THREEXUI_REQUEST_TIMEOUT environment variable.",
				Validators:  requestTimeoutValidators(),
			},
			"max_retries": schema.Int64Attribute{
				Optional:   true,
				Validators: maxRetriesValidators(),
				Description: fmt.Sprintf(
					"Maximum number of additional attempts on transient HTTP 5xx responses for idempotent write endpoints "+
						"(UpdateInbound, UpdateInboundClient, UpdateSettings, UpdateXrayTemplate, SetXrayOutboundTestURL). "+
						"Each retry waits 500ms and emits a Warn-level log so upstream flakiness is observable rather than silently absorbed. "+
						"0 disables retries entirely. Allowed range: 0..%d. Default: %d.",
					maxAllowedRetries, defaultMaxRetries,
				),
			},
		},
	}
}

func (p *ThreeXUIProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ThreeXUIProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := envString(config.Endpoint, envEndpoint, "")
	if endpoint == "" {
		resp.Diagnostics.AddError("Missing endpoint", "endpoint must be set in the provider configuration or via THREEXUI_ENDPOINT environment variable.")
		return
	}

	basePath := envString(config.BasePath, envBasePath, "/")
	username := envString(config.Username, envUsername, "admin")
	password := envString(config.Password, envPassword, "admin")
	bootstrapUsername := ""
	bootstrapUsernameSet := false
	if config.BootstrapUsername.IsUnknown() {
		resp.Diagnostics.AddError("Invalid bootstrap_username", "bootstrap_username must be known during provider configuration.")
		return
	}
	if !config.BootstrapUsername.IsNull() {
		bootstrapUsername = config.BootstrapUsername.ValueString()
		bootstrapUsernameSet = true
	}
	bootstrapPassword := ""
	bootstrapPasswordSet := false
	if config.BootstrapPassword.IsUnknown() {
		resp.Diagnostics.AddError("Invalid bootstrap_password", "bootstrap_password must be known during provider configuration.")
		return
	}
	if !config.BootstrapPassword.IsNull() {
		bootstrapPassword = config.BootstrapPassword.ValueString()
		bootstrapPasswordSet = true
	}
	if bootstrapUsernameSet != bootstrapPasswordSet {
		resp.Diagnostics.AddError(
			"Invalid bootstrap credentials",
			"bootstrap_username and bootstrap_password must be configured together.",
		)
		return
	}
	if bootstrapUsernameSet && (bootstrapUsername == "" || bootstrapPassword == "") {
		resp.Diagnostics.AddError(
			"Invalid bootstrap credentials",
			"bootstrap_username and bootstrap_password must be non-empty when configured.",
		)
		return
	}
	twoFactorCode := ""
	if !config.TwoFactorCode.IsNull() && !config.TwoFactorCode.IsUnknown() {
		twoFactorCode = config.TwoFactorCode.ValueString()
	}
	insecureSkipVerify := false
	if !config.InsecureSkipVerify.IsNull() && !config.InsecureSkipVerify.IsUnknown() {
		insecureSkipVerify = config.InsecureSkipVerify.ValueBool()
	} else if v := os.Getenv(envInsecureSkipVerify); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			resp.Diagnostics.AddError("Invalid THREEXUI_INSECURE_SKIP_VERIFY", fmt.Sprintf("must be true or false, got %q", v))
			return
		}
		insecureSkipVerify = b
	}

	timeoutStr := envString(config.RequestTimeout, envRequestTimeout, "30s")

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		resp.Diagnostics.AddError("Invalid request_timeout", err.Error())
		return
	}

	maxRetries := defaultMaxRetries
	if !config.MaxRetries.IsNull() && !config.MaxRetries.IsUnknown() {
		v := config.MaxRetries.ValueInt64()
		if v < 0 || v > maxAllowedRetries {
			resp.Diagnostics.AddError(
				"Invalid max_retries",
				fmt.Sprintf("max_retries must be between 0 and %d, got %d", maxAllowedRetries, v),
			)
			return
		}
		maxRetries = int(v)
	} else if v := os.Getenv(envMaxRetries); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid THREEXUI_MAX_RETRIES", fmt.Sprintf("must be an integer, got %q", v))
			return
		}
		if n < 0 || n > maxAllowedRetries {
			resp.Diagnostics.AddError(
				"Invalid THREEXUI_MAX_RETRIES",
				fmt.Sprintf("must be between 0 and %d, got %d", maxAllowedRetries, n),
			)
			return
		}
		maxRetries = int(n)
	}

	client, err := NewClient(ClientConfig{
		Endpoint:           endpoint,
		BasePath:           basePath,
		Username:           username,
		Password:           password,
		TwoFactorCode:      twoFactorCode,
		InsecureSkipVerify: insecureSkipVerify,
		Timeout:            timeout,
		MaxRetries:         maxRetries,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client init failed", err.Error())
		return
	}

	if bootstrapUsernameSet {
		usedBootstrap, err := client.LoginWithBootstrapCredentials(ctx, bootstrapUsername, bootstrapPassword)
		if err != nil {
			resp.Diagnostics.AddError("Login failed", err.Error())
			return
		}
		if usedBootstrap {
			resp.Diagnostics.AddWarning(
				"Bootstrap credentials used",
				"The provider authenticated with bootstrap_username/bootstrap_password for this run. Rotate the panel to the primary credentials with threexui_panel_user so future runs do not need the bootstrap credentials.",
			)
		}
	} else {
		if err := client.Login(ctx); err != nil {
			resp.Diagnostics.AddError("Login failed", err.Error())
			return
		}
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ThreeXUIProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInboundResource,
		NewNodeResource,
		NewInboundClientResource,
		NewHostGroupResource,
		NewPanelGeneralResource,
		NewPanelSecurityResource,
		NewPanelUserResource,
		NewPanelTelegramResource,
		NewPanelEmailResource,
		NewPanelDiscordResource,
		NewPanelSubscriptionResource,
		NewXrayBasicsResource,
		NewXrayDNSResource,
		NewXrayRoutingResource,
		NewXrayBalancersResource,
		NewXrayReverseResource,
		NewXrayOutboundsResource,
		NewXrayObservatoryResource,
		NewXrayVersionResource,
	}
}

func (p *ThreeXUIProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewInboundsDataSource,
		NewNodesDataSource,
		NewServerStatusDataSource,
		NewXrayVersionsDataSource,
		NewXrayConfigDataSource,
		NewSettingsDataSource,
		NewOnlineClientsDataSource,
		NewClientTrafficsDataSource,
	}
}
