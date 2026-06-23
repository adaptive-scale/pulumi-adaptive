package adaptive

import (
	"fmt"
	"strings"
)

// buildIntegrationConfig reproduces the Terraform provider's
// schemaToResourceIntegrationConfiguration dispatch: it maps a resource `type`
// to the integration-specific config struct that is marshalled to YAML and sent
// as the `config` field. It returns the config object and the effective
// integration type string sent to the API (only "services" is rewritten, to
// "servicelist").
func buildIntegrationConfig(a ResourceArgs) (any, string, error) {
	t := a.Type
	if !validIntegrationTypes[t] {
		return nil, "", fmt.Errorf("invalid integration type %q; valid types are: %s", t, validTypeList())
	}

	var cfg any
	switch t {
	case "aws":
		cfg = awsConfig{Name: a.Name, Version: "1.0", AWSRegionName: sv(a.RegionName), AWSAccessKeyID: sv(a.AccessKeyID), AWSSecretAccessKey: sv(a.SecretAccessKey)}
	case "azure":
		cfg = azureConfig{Version: "1.0", Name: a.Name, TenantID: sv(a.TenantID), ApplicationID: sv(a.ApplicationID), ClientSecret: sv(a.ClientSecret)}
	case "azureactivedirectory":
		cfg = azureADConfig{Name: a.Name, Domain: sv(a.Domain), ClientID: sv(a.ClientID), ClientSecret: sv(a.ClientSecret), TenantID: sv(a.TenantID), UseTenant: bv(a.UseTenant)}
	case "awsredshift":
		cfg = awsRedshiftConfig{Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName), HostName: sv(a.Host), Port: sv(a.Port)}
	case "cockroachdb":
		cfg = cockroachConfig{Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName), HostName: sv(a.Host), Port: sv(a.Port), SSLMode: sv(a.SSLMode), RootCert: strings.TrimSpace(sv(a.TLSRootCert))}
	case "gcp":
		cfg = gcpConfig{Version: "1", Name: a.Name, ProjectID: sv(a.ProjectID), KeyFile: strings.TrimSpace(sv(a.KeyFile))}
	case "google":
		cfg = googleConfig{Version: "1", Name: a.Name, Domain: sv(a.Domain), ClientID: sv(a.ClientID), ClientSecret: sv(a.ClientSecret)}
	case "mongodb":
		cfg = mongoConfig{Name: a.Name, URI: sv(a.URI)}
	case "mysql":
		cfg = mysqlConfig{Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName), HostName: sv(a.Host), Port: sv(a.Port), SSLMode: "require"}
	case "okta":
		cfg = oktaConfig{Version: "1.0", Name: a.Name, Domain: sv(a.Domain), ClientID: sv(a.ClientID), ClientSecret: sv(a.ClientSecret)}
	case "postgres":
		cfg = postgresConfig{Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName), HostName: sv(a.Host), Port: sv(a.Port), SSLMode: sv(a.SSLMode), TLSRootCert: sv(a.TLSRootCert), TLSCertFile: sv(a.TLSCertFile), TLSKeyFile: sv(a.TLSKeyFile)}
	case "ssh":
		cfg = sshConfig{Version: "1.0", Name: a.Name, Username: sv(a.Username), UsePassword: sv(a.Key) == "", Password: sv(a.Key), HostName: sv(a.Host), Port: sv(a.Port), SSHKey: sv(a.Key)}
	case "kubernetes":
		cfg = kubernetesConfig{Name: a.Name, ApiServer: sv(a.ApiServer), ClusterToken: strings.TrimSpace(sv(a.ClusterToken)), ClusterCerts: strings.TrimSpace(sv(a.ClusterCert)), Namespace: sv(a.Namespace), TolerationsBytes: sv(a.Tolerations), AnnotationsBytes: sv(a.Annotations), NodeSelectorBytes: sv(a.NodeSelector), NodeAffinityBytes: sv(a.NodeAffinity)}
	case "awsdocumentdb":
		cfg = awsDocumentDBConfig{Name: a.Name, URI: sv(a.URI)}
	case "zerotier":
		cfg = zeroTierConfig{Name: a.Name, NetworkID: sv(a.NetworkID), Token: sv(a.APIToken), Version: "1.0"}
	case "mongodb_atlas":
		cfg = mongoAtlasConfig{Name: a.Name, URI: sv(a.URI), OrganisationID: sv(a.OrganizationID), PublicKey: sv(a.PublicKey), PrivateKey: sv(a.PrivateKey), ProjectID: sv(a.ProjectID)}
	case "rdp_windows":
		cfg = rdpWindowsConfig{Version: "1.0", Name: a.Name, Hostname: sv(a.Hostname), Password: sv(a.Password), Username: sv(a.Username), Port: sv(a.Port)}
	case "awssecretsmanager":
		cfg = awsSecretsManagerConfig{Name: a.Name, AWSRegionName: sv(a.AWSRegionName), AWSARN: sv(a.AWSArn)}
	case "postgres_aws_secrets_manager":
		cfg = secretsManagerConfig{Name: a.Name, ARN: sv(a.Arn), Region: sv(a.Region), SecretID: sv(a.SecretID)}
	case "mysql_aws_secrets_manager":
		cfg = mysqlAWSConfig{Name: a.Name, ARN: sv(a.Arn), Region: sv(a.Region), SecretID: sv(a.SecretID)}
	case "mongodb_aws_secrets_manager":
		cfg = mongoAWSConfig{Name: a.Name, ARN: sv(a.Arn), Region: sv(a.Region), SecretID: sv(a.SecretID), Key: sv(a.Key)}
	case "sql_server":
		cfg = sqlServerConfig{Name: a.Name, DatabaseName: sv(a.DatabaseName), Hostname: sv(a.Host), Port: sv(a.Port), Username: sv(a.Username), Password: sv(a.Password)}
	case "azuresqlserver":
		cfg = azureSQLServerConfig{Name: a.Name, Hostname: sv(a.Hostname), Port: sv(a.Port), Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName)}
	case "splunk":
		cfg = splunkConfig{Name: a.Name, TokenID: sv(a.TokenID), Url: sv(a.URL)}
	case "datadog":
		cfg = datadogConfig{Name: a.Name, DdSite: sv(a.DdSite), DdApiKey: sv(a.DdApiKey)}
	case "sqlserver_aws_secrets_manager":
		cfg = secretsManagerConfig{Name: a.Name, ARN: sv(a.Arn), Region: sv(a.Region), SecretID: sv(a.SecretID)}
	case "coralogix":
		cfg = coralogixConfig{Name: a.Name, Url: sv(a.URI), PrivateKey: sv(a.PrivateKey), ApplicationName: sv(a.ApplicationName), SubSystemName: sv(a.SubSystemName)}
	case "jumpcloud":
		cfg = jumpCloudConfig{Name: a.Name, ClientID: sv(a.ClientID), ClientSecret: sv(a.ClientSecret), Domain: sv(a.Domain), ApiKey: sv(a.APIToken)}
	case "msteams":
		cfg = msTeamsConfig{Name: a.Name, AppID: sv(a.ClientID), AppKey: sv(a.ClientSecret), TenantID: sv(a.TenantID)}
	case "yugabytedb":
		cfg = yugabyteConfig{Name: a.Name, Hostname: sv(a.Host), Username: sv(a.Username), Password: sv(a.Password), SSLMode: sv(a.SSLMode), RootCert: sv(a.RootCert), Port: sv(a.Port)}
	case "onelogin":
		cfg = oneLoginConfig{Name: a.Name, Domain: sv(a.Domain), ClientID: sv(a.ClientID), ClientSecret: sv(a.ClientSecret), ApiClientID: sv(a.ApiClientID), ApiClientSecret: sv(a.ApiClientSecret)}
	case "elasticsearch":
		cfg = elasticsearchConfig{Name: a.Name, Url: sv(a.URI), Username: sv(a.Username), Password: sv(a.Password), Index: sv(a.Index)}
	case "paloalto_ngfw":
		cfg = paloAltoConfig{Name: a.Name, Password: sv(a.Password), Username: sv(a.Username), Hostname: sv(a.Hostname), WebuiPort: sv(a.WebuiPort), LoginUrl: sv(a.LoginURL)}
	case "fortinet_ngfw":
		cfg = ngfwConfig{Name: a.Name, Hostname: sv(a.Hostname), LoginUrl: sv(a.URI), Port: sv(a.Port), Type: "fortinet_ngfw", UseProxy: bv(a.UseProxy), Username: sv(a.Username), Password: sv(a.Password), Version: "1.0", WebuiPort: sv(a.WebuiPort)}
	case "cisco_ngfw":
		cfg = ngfwConfig{Name: a.Name, Hostname: sv(a.Hostname), LoginUrl: sv(a.URI), Port: sv(a.Port), UseProxy: bv(a.UseProxy), Username: sv(a.Username), Password: sv(a.Password), WebuiPort: sv(a.WebuiPort)}
	case "snowflake":
		cfg = snowflakeConfig{Name: a.Name, DatabaseAccount: sv(a.Hostname), DatabaseUsername: sv(a.Username), DatabasePassword: sv(a.Password), DatabaseName: sv(a.DatabaseName), Warehouse: sv(a.Warehouse), Schema: sv(a.Schema), Clientcert: sv(a.Clientcert), Role: sv(a.Role)}
	case "snowflake_aws_secrets_manager":
		cfg = secretsManagerConfig{Name: a.Name, ARN: sv(a.Arn), Region: sv(a.Region), SecretID: sv(a.SecretID)}
	case "custom_siem_webhook":
		cfg = customSIEMWebhookConfig{Name: a.Name, Url: sv(a.URI), SharedSecret: sv(a.SharedSecret)}
	case "aruba_sw":
		cfg = arubaSWConfig{Name: a.Name, Hostname: sv(a.Hostname), Username: sv(a.Username), Password: sv(a.Password)}
	case "aruba_instant_on":
		cfg = arubaInstantOnConfig{Name: a.Name, Host: sv(a.Host), Port: sv(a.Port), Username: sv(a.Username), Password: sv(a.Password), APIToken: sv(a.APIToken)}
	case "hpe_switch":
		cfg = ngfwConfig{Name: a.Name, Hostname: sv(a.Hostname), LoginUrl: sv(a.URI), Port: sv(a.Port), UseProxy: bv(a.UseProxy), Username: sv(a.Username), Password: sv(a.Password), WebuiPort: sv(a.WebuiPort)}
	case "syslog":
		cfg = syslogConfig{Name: a.Name, Hostname: sv(a.Hostname), Port: sv(a.Port), Protocol: sv(a.Protocol)}
	case "customintegration":
		cfg = customIntegrationConfig{Name: a.Name, Image: sv(a.Image), ServiceAccountName: sv(a.ServiceAccountName)}
	case "clickhouse":
		cfg = clickhouseConfig{Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName), HostName: sv(a.Host), Port: sv(a.Port), SSLMode: sv(a.SSLMode)}
	case "keyspaces":
		cfg = keyspacesConfig{UseServiceAccount: bv(a.UseServiceAccount), CreateIfNotExists: bv(a.CreateIfNotExists), Name: a.Name}
	case "rabbitmq":
		cfg = rabbitMQConfig{Url: sv(a.URI), Name: a.Name, Username: sv(a.Username), Password: sv(a.Password)}
	case "azurecosmosnosql":
		cfg = azureCosmosConfig{Name: a.Name, Endpoint: sv(a.URI), Key: sv(a.APIToken)}
	case "services":
		cfg = serviceListConfig{Version: "1", Name: a.Name, URLs: sv(a.URLs)}
	case "serverlist":
		cfg = serverListConfig{Version: "1", Hosts: strings.Join(a.Hosts, "\n"), DefaultUser: sv(a.DefaultUser), SshKey: sv(a.Key), Password: sv(a.Password)}
	case "msteams_workflow":
		cfg = msTeamsWorkflowConfig{Name: a.Name, WebhookURL: sv(a.WebhookURL)}
	default:
		return nil, "", fmt.Errorf("unsupported integration type %q", t)
	}

	effectiveType := t
	if t == "services" {
		effectiveType = "servicelist"
	}
	return cfg, effectiveType, nil
}

// ===========================================================================
// Integration config structs (yaml tags must match the Adaptive API exactly).
// ===========================================================================

type awsConfig struct {
	Name               string `yaml:"name"`
	Version            string `yaml:"version"`
	AWSRegionName      string `yaml:"aws_region_name"`
	AWSAccessKeyID     string `yaml:"aws_access_key_id"`
	AWSSecretAccessKey string `yaml:"aws_secret_access_key"`
}

type azureConfig struct {
	Version       string `yaml:"version"`
	Name          string `yaml:"name"`
	TenantID      string `yaml:"tenantID"`
	ApplicationID string `yaml:"applicationID"`
	ClientSecret  string `yaml:"clientSecret"`
}

type azureADConfig struct {
	Name         string `yaml:"name"`
	Domain       string `yaml:"domain"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
	TenantID     string `yaml:"tenantID"`
	UseTenant    bool   `yaml:"useTenant"`
}

type awsRedshiftConfig struct {
	Version      string `yaml:"version"`
	Name         string `yaml:"name"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DatabaseName string `yaml:"databaseName"`
	HostName     string `yaml:"hostname"`
	Port         string `yaml:"port"`
	SSLMode      string `yaml:"sslMode"`
}

type cockroachConfig struct {
	Name         string `yaml:"name"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DatabaseName string `yaml:"databaseName"`
	HostName     string `yaml:"hostname"`
	Port         string `yaml:"port"`
	SSLMode      string `yaml:"sslMode"`
	RootCert     string `yaml:"rootCert"`
}

type gcpConfig struct {
	Version   string `yaml:"version"`
	Name      string `yaml:"name"`
	ProjectID string `yaml:"project_id"`
	KeyFile   string `yaml:"key_file"`
}

type googleConfig struct {
	Version      string `yaml:"Version"`
	Name         string `yaml:"name"`
	Domain       string `yaml:"domain"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
}

type mongoConfig struct {
	Name string `yaml:"name"`
	URI  string `yaml:"uri"`
}

type mysqlConfig struct {
	Version      string `yaml:"version"`
	Name         string `yaml:"name"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DatabaseName string `yaml:"databaseName"`
	HostName     string `yaml:"hostname"`
	Port         string `yaml:"port"`
	SSLMode      string `yaml:"sslMode"`
}

type oktaConfig struct {
	Version      string `yaml:"version"`
	Name         string `yaml:"name"`
	Domain       string `yaml:"domain"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
}

type postgresConfig struct {
	Name         string `yaml:"name"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DatabaseName string `yaml:"databaseName"`
	HostName     string `yaml:"hostname"`
	Port         string `yaml:"port"`
	SSLMode      string `yaml:"sslMode"`
	TLSRootCert  string `yaml:"rootCert"`
	TLSCertFile  string `yaml:"crtText"`
	TLSKeyFile   string `yaml:"keyText"`
}

type sshConfig struct {
	Version     string `yaml:"version"`
	Name        string `yaml:"name"`
	Username    string `yaml:"username"`
	UsePassword bool   `yaml:"usePassword"`
	Password    string `yaml:"password"`
	HostName    string `yaml:"hostname"`
	Port        string `yaml:"port"`
	SSHKey      string `yaml:"sshKey"`
}

type kubernetesConfig struct {
	Name              string `yaml:"name"`
	ApiServer         string `yaml:"apiserver"`
	ClusterToken      string `yaml:"token"`
	ClusterCerts      string `yaml:"cacrt"`
	Namespace         string `yaml:"namespace,omitempty"`
	TolerationsBytes  string `yaml:"tolerationsBytes,omitempty"`
	AnnotationsBytes  string `yaml:"annotationsBytes,omitempty"`
	NodeSelectorBytes string `yaml:"nodeSelectorBytes,omitempty"`
	NodeAffinityBytes string `yaml:"affinityBytes,omitempty"`
}

type awsDocumentDBConfig struct {
	Name string `yaml:"name"`
	URI  string `yaml:"uri"`
}

type zeroTierConfig struct {
	Name      string `yaml:"name"`
	NetworkID string `yaml:"network_id"`
	Token     string `yaml:"api_token,omitempty"`
	Version   string `yaml:"version,omitempty"`
}

type mongoAtlasConfig struct {
	Name           string `yaml:"name"`
	URI            string `yaml:"uri"`
	OrganisationID string `yaml:"organization_id"`
	PublicKey      string `yaml:"public_key"`
	PrivateKey     string `yaml:"private_key"`
	ProjectID      string `yaml:"project_id"`
}

type rdpWindowsConfig struct {
	Version  string `yaml:"version"`
	Name     string `yaml:"name"`
	Hostname string `yaml:"hostname"`
	Password string `yaml:"password"`
	Username string `yaml:"username"`
	Port     string `yaml:"port"`
}

type awsSecretsManagerConfig struct {
	Version       string `yaml:"version"`
	Name          string `yaml:"name"`
	AWSRegionName string `yaml:"aws_region_name"`
	AWSARN        string `yaml:"aws_arn"`
}

// secretsManagerConfig is shared by the postgres/sqlserver/snowflake AWS
// Secrets Manager integrations, which all use the same arn/region/secret_id shape.
type secretsManagerConfig struct {
	Name     string `yaml:"name"`
	ARN      string `yaml:"arn"`
	Region   string `yaml:"region"`
	SecretID string `yaml:"secret_id"`
}

type mysqlAWSConfig struct {
	Version  string `yaml:"version"`
	Name     string `yaml:"name"`
	ARN      string `yaml:"arn"`
	Region   string `yaml:"region"`
	SecretID string `yaml:"secret_id"`
}

type mongoAWSConfig struct {
	Version  string `yaml:"version"`
	Name     string `yaml:"name"`
	ARN      string `yaml:"arn"`
	Region   string `yaml:"region"`
	SecretID string `yaml:"secret_id"`
	Key      string `yaml:"key"`
}

type sqlServerConfig struct {
	Name         string `yaml:"name"`
	DatabaseName string `yaml:"databaseName"`
	Hostname     string `yaml:"hostname"`
	Port         string `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

type azureSQLServerConfig struct {
	Name         string `yaml:"name"`
	Hostname     string `yaml:"hostname"`
	Port         string `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DatabaseName string `yaml:"databaseName"`
}

type splunkConfig struct {
	Name    string `yaml:"name"`
	TokenID string `yaml:"tokenID"`
	Url     string `yaml:"url"`
}

type datadogConfig struct {
	Name     string `yaml:"name"`
	DdSite   string `yaml:"dd_site"`
	DdApiKey string `yaml:"dd_api_key"`
}

type coralogixConfig struct {
	Name            string `yaml:"name"`
	Url             string `yaml:"url"`
	PrivateKey      string `yaml:"privateKey"`
	ApplicationName string `yaml:"applicationName"`
	SubSystemName   string `yaml:"subSystemName"`
}

type jumpCloudConfig struct {
	Name         string `yaml:"name"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
	Domain       string `yaml:"domain"`
	ApiKey       string `yaml:"apiKey"`
}

type msTeamsConfig struct {
	Name     string `yaml:"name"`
	AppID    string `yaml:"appID"`
	AppKey   string `yaml:"appKey"`
	TenantID string `yaml:"tenantID"`
}

type yugabyteConfig struct {
	Name     string `yaml:"name"`
	Hostname string `yaml:"hostname"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"sslMode"`
	RootCert string `yaml:"rootCert"`
	Port     string `yaml:"port"`
}

type oneLoginConfig struct {
	Name            string `yaml:"name"`
	Domain          string `yaml:"domain"`
	ClientID        string `yaml:"clientID"`
	ClientSecret    string `yaml:"clientSecret"`
	ApiClientID     string `yaml:"apiClientID"`
	ApiClientSecret string `yaml:"apiClientSecret"`
}

type elasticsearchConfig struct {
	Name     string `yaml:"name"`
	Url      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Index    string `yaml:"index"`
}

type paloAltoConfig struct {
	Name      string `yaml:"name"`
	Password  string `yaml:"password"`
	Username  string `yaml:"username"`
	Hostname  string `yaml:"hostname"`
	WebuiPort string `yaml:"webui_port"`
	LoginUrl  string `yaml:"login_url"`
}

// ngfwConfig is shared by fortinet_ngfw, cisco_ngfw, and hpe_switch (identical shape).
type ngfwConfig struct {
	Name      string `yaml:"name"`
	Hostname  string `yaml:"hostname"`
	LoginUrl  string `yaml:"login_url"`
	Port      string `yaml:"port"`
	Type      string `yaml:"type"`
	UseProxy  bool   `yaml:"use_proxy"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Version   string `yaml:"version"`
	WebuiPort string `yaml:"webui_port"`
}

type snowflakeConfig struct {
	Name             string `yaml:"name"`
	DatabaseAccount  string `yaml:"databaseAccount"`
	DatabaseUsername string `yaml:"databaseUsername"`
	DatabasePassword string `yaml:"databasePassword"`
	DatabaseName     string `yaml:"databaseName"`
	Warehouse        string `yaml:"warehouse"`
	Schema           string `yaml:"schema"`
	Clientcert       string `yaml:"clientcert"`
	Role             string `yaml:"role"`
}

type customSIEMWebhookConfig struct {
	Name         string `yaml:"name"`
	Url          string `yaml:"url"`
	SharedSecret string `yaml:"sharedSecret,omitempty"`
}

type arubaSWConfig struct {
	Name     string `yaml:"name"`
	Hostname string `yaml:"hostname"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type arubaInstantOnConfig struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	APIToken string `yaml:"apiToken"`
}

type syslogConfig struct {
	Name     string `yaml:"name"`
	Hostname string `yaml:"hostname"`
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
}

type customIntegrationConfig struct {
	Name               string `yaml:"name"`
	Image              string `yaml:"image"`
	ServiceAccountName string `yaml:"service_account_name,omitempty"`
}

type clickhouseConfig struct {
	Name         string `yaml:"name"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	DatabaseName string `yaml:"databaseName"`
	HostName     string `yaml:"hostname"`
	Port         string `yaml:"port"`
	SSLMode      string `yaml:"sslMode"`
}

type keyspacesConfig struct {
	UseServiceAccount bool   `yaml:"use_service_account"`
	CreateIfNotExists bool   `yaml:"create_if_not_exists"`
	Name              string `yaml:"name"`
}

type rabbitMQConfig struct {
	Url      string `yaml:"url"`
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type azureCosmosConfig struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
	Key      string `yaml:"key"`
}

type serviceListConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	URLs    string `yaml:"urls"`
}

type serverListConfig struct {
	Version     string `yaml:"version"`
	Hosts       string `yaml:"hosts"`
	DefaultUser string `yaml:"user"`
	SshKey      string `yaml:"sshKey"`
	Password    string `yaml:"password"`
}

// ===========================================================================
// Valid integration types (port of the Terraform provider's validIntegrationTypes).
// ===========================================================================

var validIntegrationTypes = map[string]bool{
	"aws": true, "azure": true, "azureactivedirectory": true, "cockroachdb": true,
	"gcp": true, "google": true, "mongodb": true, "mongodb_aws_secrets_manager": true,
	"mysql": true, "mysql_aws_secrets_manager": true, "okta": true, "postgres": true,
	"postgres_aws_secrets_manager": true, "services": true, "serverlist": true, "ssh": true,
	"kubernetes": true, "awsdocumentdb": true, "awsredshift": true, "zerotier": true,
	"rdp_windows": true, "mongodb_atlas": true, "awssecretsmanager": true, "sql_server": true,
	"azuresqlserver": true, "splunk": true, "datadog": true, "sqlserver_aws_secrets_manager": true,
	"coralogix": true, "jumpcloud": true, "msteams": true, "yugabytedb": true, "onelogin": true,
	"elasticsearch": true, "paloalto_ngfw": true, "fortinet_ngfw": true, "cisco_ngfw": true,
	"snowflake": true, "snowflake_aws_secrets_manager": true, "custom_siem_webhook": true,
	"aruba_sw": true, "aruba_instant_on": true, "hpe_switch": true, "syslog": true,
	"customintegration": true, "clickhouse": true, "keyspaces": true, "rabbitmq": true,
	"azurecosmosnosql": true, "msteams_workflow": true,
}

func validTypeList() string {
	types := make([]string, 0, len(validIntegrationTypes))
	for t := range validIntegrationTypes {
		types = append(types, t)
	}
	return strings.Join(types, ", ")
}
