package adaptive

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
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
	case "1password":
		cfg = onePasswordConfig{Name: a.Name, ServiceAccount: sv(a.ServiceAccount), UseConnectServer: bv(a.UseConnectServer), ConnectServerURL: sv(a.ConnectServerURL)}
	case "adaptive_rdp":
		cfg = adaptiveRDPConfig{Version: "1.0", Name: a.Name, Targets: sv(a.Targets)}
	case "adaptiveencrypt":
		cfg = adaptiveEncryptConfig{Name: a.Name, Value: sv(a.Value)}
	case "adaptiveremotedesktop":
		cfg = adaptiveRemoteDesktopConfig{Version: "1.0", Name: a.Name, Cpu: sv(a.CPU), Memory: sv(a.Memory), Storage: sv(a.Storage)}
	case "avastbusinessedr":
		cfg = avastBusinessEDRConfig{Name: a.Name, ClientID: sv(a.ClientID), ClientSecret: sv(a.ClientSecret)}
	case "awscloudwatchlogs":
		cfg = awsCloudWatchLogsConfig{Version: "1.0", Name: a.Name, AWSRegion: sv(a.AWSRegionName), ARN: sv(a.Arn), LogGroupName: sv(a.LogGroupName), LogStreamName: sv(a.LogStreamName)}
	case "awsdocumentdb_aws_secret_manager":
		cfg = documentDBSecretsManagerConfig{Version: "1.0", Name: a.Name, ARN: sv(a.Arn), Region: sv(a.Region), SecretID: sv(a.SecretID), TlsEnabled: bv(a.TLSEnabled)}
	case "awselasticcache":
		cfg = elastiCacheConfig{Version: "1.0", Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), Host: sv(a.Host), Port: sv(a.Port), TlsEnabled: bv(a.TLSEnabled), TlsSkipVerify: bv(a.TLSSkipVerify), AccessControlMethod: sv(a.AccessControlMethod), AccessControlGroupID: sv(a.AccessControlGroup), AWSRegionName: sv(a.AWSRegionName), AWSAccessKeyID: sv(a.AccessKeyID), AWSSecretAccessKey: sv(a.SecretAccessKey)}
	case "azure_documentdb":
		cfg = azureDocumentDBConfig{Version: "1.0", Name: a.Name, URI: sv(a.URI)}
	case "big_query":
		cfg = bigQueryConfig{Version: "1.0", Name: a.Name, CredentialJSON: strings.TrimSpace(sv(a.CredentialJSON))}
	case "chrome":
		cfg = chromeConfig{Name: a.Name, URL: sv(a.URL), Prestart: sv(a.Prestart), AutomationMode: sv(a.AutomationMode), Fields: sv(a.Fields), Script: sv(a.Script)}
	case "cockroachdb_aws_secrets_manager":
		cfg = cockroachSecretsManagerConfig{Name: a.Name, ARN: sv(a.Arn), Region: sv(a.Region), SecretID: sv(a.SecretID), Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName), HostName: sv(a.Host), Port: sv(a.Port), RootCert: strings.TrimSpace(sv(a.RootCert)), SSLMode: sv(a.SSLMode)}
	case "confluent":
		cfg = confluentConfig{Version: "1.0", Name: a.Name, OrganizationID: sv(a.OrganizationID), Username: sv(a.Username), Password: sv(a.Password), Resource: sv(a.Resource), APIKey: sv(a.APIKey), APISecret: sv(a.APISecret)}
	case "digitalocean":
		cfg = digitalOceanConfig{Version: "1.0", Name: a.Name, ApiToken: sv(a.APIToken)}
	case "fortinet_analyzer", "fortinet_manager":
		cfg = fortinetApplianceConfig{Version: "1.0", Name: a.Name, Username: sv(a.Username), UsePassword: sv(a.Key) == "", Password: sv(a.Password), HostName: sv(a.Hostname), Port: sv(a.Port), WebUIPort: sv(a.WebuiPort), LoginURL: sv(a.LoginURL), SSHKey: sv(a.Key)}
	case "heroku":
		cfg = herokuConfig{Name: a.Name, Machine: sv(a.Machine), Username: sv(a.Username), Token: sv(a.Token)}
	case "ivanti":
		cfg = ivantiConfig{Version: "1.0", Name: a.Name, Username: sv(a.Username), UsePassword: sv(a.Key) == "", Password: sv(a.Password), HostName: sv(a.Hostname), Port: sv(a.Port), WebUIPort: sv(a.WebuiPort), LoginURL: sv(a.LoginURL), UseProxy: bv(a.UseProxy), SSHKey: sv(a.Key)}
	case "juniper_sw", "sophos_fw":
		cfg = sshApplianceConfig{Version: "1.0", Name: a.Name, Username: sv(a.Username), UsePassword: sv(a.Key) == "", Password: sv(a.Password), HostName: sv(a.Hostname), Port: sv(a.Port), SSHKey: sv(a.Key)}
	case "kafka":
		cfg = kafkaConfig{Name: a.Name, ClientConfiguration: sv(a.ClientConfiguration)}
	case "ldap":
		cfg = ldapConfig{Name: a.Name, Hostname: sv(a.Hostname), Port: sv(a.Port), EncryptionMethod: sv(a.LdapEncryptionMethod), SearchBindDN: sv(a.LdapSearchBindDN), SearchBindPassword: sv(a.LdapSearchBindPassword), UserNameAttribute: sv(a.LdapUserNameAttribute), UserBaseDN: sv(a.LdapUserBaseDN)}
	case "mongodb-do":
		cfg = mongoDBDOConfig{Version: "1.0", Name: a.Name, DBName: sv(a.DatabaseName), APIToken: sv(a.APIToken)}
	case "mongodb36":
		cfg = mongo36Config{Version: "1.0", Name: a.Name, URI: sv(a.URI), ClientCertificate: strings.TrimSpace(sv(a.ClientCertificate)), UseTLS: bv(a.UseTLS)}
	case "oracle":
		cfg = oracleConfig{Name: a.Name, HostName: sv(a.Host), Port: sv(a.Port), Username: sv(a.Username), Password: sv(a.Password), ServiceName: sv(a.ServiceName)}
	case "proxysql":
		cfg = proxysqlConfig{Version: "1.0", Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), DatabaseName: sv(a.DatabaseName), HostName: sv(a.Host), Port: sv(a.Port), SSLMode: sv(a.SSLMode), RootCert: strings.TrimSpace(sv(a.RootCert)), ClientCert: strings.TrimSpace(sv(a.ClientCert)), ClientKey: strings.TrimSpace(sv(a.ClientKey)), OldVersion: bv(a.OldVersion), ProxysqlAdminPort: sv(a.ProxysqlAdminPort), ProxysqlAdminUsername: sv(a.ProxysqlAdminUsername), ProxysqlAdminPassword: sv(a.ProxysqlAdminPassword), ProxysqlHostgroupID: sv(a.ProxysqlHostgroupID)}
	case "rdpldap":
		cfg = rdpLdapConfig{Version: "1.0", Name: a.Name, Hostname: sv(a.Hostname), Port: sv(a.Port), LdapHost: sv(a.LdapHostname), LdapPort: sv(a.LdapPort), EncryptionMethod: sv(a.LdapEncryptionMethod), SearchBindDN: sv(a.LdapSearchBindDN), SearchBindPassword: sv(a.LdapSearchBindPassword), UserBaseDN: sv(a.LdapUserBaseDN), UserNameAttribute: sv(a.LdapUserNameAttribute)}
	case "redis":
		cfg = redisConfig{Version: "1.0", Name: a.Name, Username: sv(a.Username), Password: sv(a.Password), Host: sv(a.Host), Port: sv(a.Port), TlsEnabled: bv(a.TLSEnabled), TlsSkipVerify: bv(a.TLSSkipVerify), IsRedisLabs: bv(a.IsRedisLabs), CertFile: strings.TrimSpace(sv(a.TLSCertFile)), KeyFile: strings.TrimSpace(sv(a.TLSKeyFile)), CAFile: strings.TrimSpace(sv(a.TLSCACert))}
	case "securetunnels":
		cfg = secureTunnelsConfig{Name: a.Name}
	case "slack_webhook":
		cfg = slackWebhookConfig{Version: "1.0", Name: a.Name, WebhookURL: sv(a.WebhookURL), APIToken: sv(a.APIToken)}
	case "vnc":
		cfg = vncConfig{Name: a.Name, Hosts: strings.Join(a.Hosts, "\n"), Port: sv(a.Port), Password: sv(a.Password)}
	case "windows-server-groups":
		cfg = windowsServerGroupsConfig{Version: "1.0", Name: a.Name, Hosts: strings.Join(a.Hosts, "\n"), Username: sv(a.Username), Password: sv(a.Password), Port: sv(a.Port)}
	default:
		return nil, "", fmt.Errorf("unsupported integration type %q", t)
	}

	return cfg, wireType(t), nil
}

// wireType maps the program-facing `type` onto the integration type string the
// API stores. Two differ: the registry keys the service list as "servicelist",
// and ZeroTier as "zero_tier" (both types.ZeroTier and the integrations-table
// row use that spelling, so a resource stored as "zerotier" resolves to no
// implementation server-side and is dropped from the resource listing join).
func wireType(t string) string {
	switch t {
	case "services":
		return "servicelist"
	case "zerotier":
		return "zero_tier"
	}
	return t
}

// providerType is the inverse of wireType: the `type` a program writes for the
// integration type the API reports.
func providerType(w string) string {
	switch w {
	case "servicelist":
		return "services"
	case "zero_tier":
		return "zerotier"
	}
	return w
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

type msTeamsWorkflowConfig struct {
	Name       string `yaml:"name"`
	WebhookURL string `yaml:"webhookURL"`
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

type onePasswordConfig struct {
	Name             string `yaml:"name"`
	ServiceAccount   string `yaml:"service_account"`
	UseConnectServer bool   `yaml:"use_connect_server"`
	ConnectServerURL string `yaml:"connect_server_url"`
}

type adaptiveRDPConfig struct {
	Version string `yaml:"version"`
	Name    string `yaml:"name"`
	Targets string `yaml:"targets"`
}

type adaptiveEncryptConfig struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// adaptiveRemoteDesktopConfig uses the lowercase keys remote_desktop's
// IntegrationConfiguration unmarshals (`cpu`/`memory`/`storage`), not the
// capitalised form labels — yaml.v2 matches struct tags case-sensitively.
type adaptiveRemoteDesktopConfig struct {
	Version string `yaml:"version"`
	Name    string `yaml:"name"`
	Cpu     string `yaml:"cpu"`
	Memory  string `yaml:"memory"`
	Storage string `yaml:"storage"`
}

type avastBusinessEDRConfig struct {
	Name         string `yaml:"name"`
	ClientID     string `yaml:"clientID"`
	ClientSecret string `yaml:"clientSecret"`
}

type awsCloudWatchLogsConfig struct {
	Version       string `yaml:"version"`
	Name          string `yaml:"name"`
	AWSRegion     string `yaml:"aws_region"`
	ARN           string `yaml:"arn"`
	LogGroupName  string `yaml:"log_group_name"`
	LogStreamName string `yaml:"log_stream_name"`
}

type documentDBSecretsManagerConfig struct {
	Version    string `yaml:"version"`
	Name       string `yaml:"name"`
	ARN        string `yaml:"arn"`
	Region     string `yaml:"region"`
	SecretID   string `yaml:"secret_id"`
	TlsEnabled bool   `yaml:"tlsEnabled"`
}

type elastiCacheConfig struct {
	Version              string `yaml:"version"`
	Name                 string `yaml:"name"`
	Username             string `yaml:"username"`
	Password             string `yaml:"password"`
	Host                 string `yaml:"host"`
	Port                 string `yaml:"port"`
	TlsEnabled           bool   `yaml:"tlsEnabled"`
	TlsSkipVerify        bool   `yaml:"tlsSkipVerify"`
	AccessControlMethod  string `yaml:"access_control_method"`
	AccessControlGroupID string `yaml:"access_control_group"`
	AWSRegionName        string `yaml:"aws_region_name"`
	AWSAccessKeyID       string `yaml:"aws_access_key_id"`
	AWSSecretAccessKey   string `yaml:"aws_secret_access_key"`
}

type azureDocumentDBConfig struct {
	Version string `yaml:"version"`
	Name    string `yaml:"name"`
	URI     string `yaml:"uri"`
}

type bigQueryConfig struct {
	Version        string `yaml:"version"`
	Name           string `yaml:"name"`
	CredentialJSON string `yaml:"credential_json"`
}

type chromeConfig struct {
	Name           string `yaml:"name"`
	URL            string `yaml:"url"`
	Prestart       string `yaml:"prestart,omitempty"`
	AutomationMode string `yaml:"automationMode,omitempty"`
	Fields         string `yaml:"fields,omitempty"`
	Script         string `yaml:"script,omitempty"`
}

// cockroachSecretsManagerConfig carries the db_* overrides the integration reads
// in addition to the four fields its form collects; they stay out of the blob
// unless set.
type cockroachSecretsManagerConfig struct {
	Name         string `yaml:"name"`
	ARN          string `yaml:"arn"`
	Region       string `yaml:"region"`
	SecretID     string `yaml:"secret_id"`
	Username     string `yaml:"db_user,omitempty"`
	Password     string `yaml:"db_password,omitempty"`
	DatabaseName string `yaml:"db_name,omitempty"`
	HostName     string `yaml:"db_host,omitempty"`
	Port         string `yaml:"db_port,omitempty"`
	RootCert     string `yaml:"db_rootCert,omitempty"`
	SSLMode      string `yaml:"sslMode,omitempty"`
}

type confluentConfig struct {
	Version        string `yaml:"version"`
	Name           string `yaml:"name"`
	OrganizationID string `yaml:"organization_id"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	Resource       string `yaml:"resource"`
	APIKey         string `yaml:"apiKey"`
	APISecret      string `yaml:"apiSecret"`
}

type digitalOceanConfig struct {
	Version  string `yaml:"version"`
	Name     string `yaml:"name"`
	ApiToken string `yaml:"api_token"`
}

// fortinetApplianceConfig is shared by fortinet_analyzer and fortinet_manager
// (identical shape).
type fortinetApplianceConfig struct {
	Version     string `yaml:"version"`
	Name        string `yaml:"name"`
	Username    string `yaml:"username"`
	UsePassword bool   `yaml:"usePassword"`
	Password    string `yaml:"password"`
	HostName    string `yaml:"hostname"`
	Port        string `yaml:"port"`
	WebUIPort   string `yaml:"webui_port"`
	LoginURL    string `yaml:"login_url"`
	SSHKey      string `yaml:"sshKey"`
}

type herokuConfig struct {
	Name     string `yaml:"name"`
	Machine  string `yaml:"machine"`
	Username string `yaml:"username"`
	Token    string `yaml:"token"`
}

type ivantiConfig struct {
	Version     string `yaml:"version"`
	Name        string `yaml:"name"`
	Username    string `yaml:"username"`
	UsePassword bool   `yaml:"usePassword"`
	Password    string `yaml:"password"`
	HostName    string `yaml:"hostname"`
	Port        string `yaml:"port"`
	WebUIPort   string `yaml:"webui_port"`
	LoginURL    string `yaml:"login_url"`
	UseProxy    bool   `yaml:"use_proxy"`
	SSHKey      string `yaml:"sshKey"`
}

// sshApplianceConfig is shared by juniper_sw and sophos_fw (identical shape).
type sshApplianceConfig struct {
	Version     string `yaml:"version"`
	Name        string `yaml:"name"`
	Username    string `yaml:"username"`
	UsePassword bool   `yaml:"usePassword"`
	Password    string `yaml:"password"`
	HostName    string `yaml:"hostname"`
	Port        string `yaml:"port"`
	SSHKey      string `yaml:"sshKey"`
}

type kafkaConfig struct {
	Name                string `yaml:"name"`
	ClientConfiguration string `yaml:"client_configuration"`
}

type ldapConfig struct {
	Name               string `yaml:"name"`
	Hostname           string `yaml:"hostname"`
	Port               string `yaml:"port"`
	EncryptionMethod   string `yaml:"ldap-encryption-method"`
	SearchBindDN       string `yaml:"ldap-search-bind-dn"`
	SearchBindPassword string `yaml:"ldap-search-bind-password"`
	UserNameAttribute  string `yaml:"ldap-user-name-attribute"`
	UserBaseDN         string `yaml:"ldap-user-base-dn"`
}

type mongoDBDOConfig struct {
	Version  string `yaml:"version"`
	Name     string `yaml:"name"`
	DBName   string `yaml:"dbname"`
	APIToken string `yaml:"api_token"`
}

type mongo36Config struct {
	Version           string `yaml:"version"`
	Name              string `yaml:"name"`
	URI               string `yaml:"uri"`
	ClientCertificate string `yaml:"clientCertificate,omitempty"`
	UseTLS            bool   `yaml:"useTLS,omitempty"`
}

type oracleConfig struct {
	Name        string `yaml:"name"`
	HostName    string `yaml:"hostname"`
	Port        string `yaml:"port"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	ServiceName string `yaml:"service_name"`
}

type proxysqlConfig struct {
	Version               string `yaml:"version"`
	Name                  string `yaml:"name"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	DatabaseName          string `yaml:"databaseName"`
	HostName              string `yaml:"hostname"`
	Port                  string `yaml:"port"`
	SSLMode               string `yaml:"sslMode"`
	RootCert              string `yaml:"rootCert"`
	ClientCert            string `yaml:"clientCert"`
	ClientKey             string `yaml:"clientKey"`
	OldVersion            bool   `yaml:"oldVersion"`
	ProxysqlAdminPort     string `yaml:"proxysqlAdminPort"`
	ProxysqlAdminUsername string `yaml:"proxysqlAdminUsername"`
	ProxysqlAdminPassword string `yaml:"proxysqlAdminPassword"`
	ProxysqlHostgroupID   string `yaml:"proxysqlHostgroupID"`
}

type rdpLdapConfig struct {
	Version            string `yaml:"version"`
	Name               string `yaml:"name"`
	Hostname           string `yaml:"hostname"`
	Port               string `yaml:"port"`
	LdapHost           string `yaml:"ldap-hostname"`
	LdapPort           string `yaml:"ldap-port"`
	EncryptionMethod   string `yaml:"ldap-encryption-method"`
	SearchBindDN       string `yaml:"ldap-search-bind-dn"`
	SearchBindPassword string `yaml:"ldap-search-bind-password"`
	UserBaseDN         string `yaml:"ldap-user-base-dn"`
	UserNameAttribute  string `yaml:"ldap-user-name-attribute"`
}

type redisConfig struct {
	Version       string `yaml:"version"`
	Name          string `yaml:"name"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	Host          string `yaml:"host"`
	Port          string `yaml:"port"`
	TlsEnabled    bool   `yaml:"tlsEnabled"`
	TlsSkipVerify bool   `yaml:"tlsSkipVerify"`
	IsRedisLabs   bool   `yaml:"isRedisLabs"`
	CertFile      string `yaml:"crtText"`
	KeyFile       string `yaml:"keyText"`
	CAFile        string `yaml:"caText"`
}

type secureTunnelsConfig struct {
	Name string `yaml:"name"`
}

type slackWebhookConfig struct {
	Version    string `yaml:"version"`
	Name       string `yaml:"name"`
	WebhookURL string `yaml:"webhookURL"`
	APIToken   string `yaml:"apiToken"`
}

type vncConfig struct {
	Name     string `yaml:"name"`
	Hosts    string `yaml:"hosts"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}

type windowsServerGroupsConfig struct {
	Version  string `yaml:"version"`
	Name     string `yaml:"name"`
	Hosts    string `yaml:"hosts"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Port     string `yaml:"port"`
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
	"azurecosmosnosql": true, "msteams_workflow": true, "1password": true,
	"adaptive_rdp": true, "adaptiveencrypt": true, "adaptiveremotedesktop": true,
	"avastbusinessedr": true, "awscloudwatchlogs": true, "awselasticcache": true,
	"awsdocumentdb_aws_secret_manager": true, "azure_documentdb": true, "big_query": true,
	"chrome": true, "cockroachdb_aws_secrets_manager": true, "confluent": true,
	"digitalocean": true, "fortinet_analyzer": true, "fortinet_manager": true,
	"heroku": true, "ivanti": true, "juniper_sw": true, "kafka": true, "ldap": true,
	"mongodb-do": true, "mongodb36": true, "oracle": true, "proxysql": true,
	"rdpldap": true, "redis": true, "securetunnels": true, "slack_webhook": true,
	"sophos_fw": true, "vnc": true, "windows-server-groups": true,
}

func validTypeList() string {
	types := make([]string, 0, len(validIntegrationTypes))
	for t := range validIntegrationTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}

// ===========================================================================
// Reverse mapping: server configuration -> ResourceArgs.
// ===========================================================================

// applyIntegrationConfig is the inverse of buildIntegrationConfig. It folds the
// configuration the read endpoint returned back onto the arguments a program
// wrote.
//
// Secrets never appear in cfg — the server strips them and names them in
// RedactedKeys — so a key that is absent must leave its argument untouched
// rather than clear it. Keys the forward mapping derives (usePassword, the
// hardcoded version/type constants, mysql's forced sslMode) are skipped: they
// carry no user intent to reconcile.
func applyIntegrationConfig(a *ResourceArgs, integrationType string, cfg map[string]any) {
	if len(cfg) == 0 {
		return
	}
	switch providerType(integrationType) {
	case "aws":
		setStr(&a.RegionName, cfg, "aws_region_name")
		setStr(&a.AccessKeyID, cfg, "aws_access_key_id")
		setStr(&a.SecretAccessKey, cfg, "aws_secret_access_key")
	case "azure":
		setStr(&a.TenantID, cfg, "tenantID")
		setStr(&a.ApplicationID, cfg, "applicationID")
		setStr(&a.ClientSecret, cfg, "clientSecret")
	case "azureactivedirectory":
		setStr(&a.Domain, cfg, "domain")
		setStr(&a.ClientID, cfg, "clientID")
		setStr(&a.ClientSecret, cfg, "clientSecret")
		setStr(&a.TenantID, cfg, "tenantID")
		setBool(&a.UseTenant, cfg, "useTenant")
	case "awsredshift":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
	case "cockroachdb":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.SSLMode, cfg, "sslMode")
		setStr(&a.TLSRootCert, cfg, "rootCert")
	case "gcp":
		setStr(&a.ProjectID, cfg, "project_id")
		setStr(&a.KeyFile, cfg, "key_file")
	case "google":
		setStr(&a.Domain, cfg, "domain")
		setStr(&a.ClientID, cfg, "clientID")
		setStr(&a.ClientSecret, cfg, "clientSecret")
	case "mongodb":
		setStr(&a.URI, cfg, "uri")
	case "mysql":
		// sslMode is forced to "require" by the forward mapping; it is not an input.
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
	case "okta":
		setStr(&a.Domain, cfg, "domain")
		setStr(&a.ClientID, cfg, "clientID")
		setStr(&a.ClientSecret, cfg, "clientSecret")
	case "postgres":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.SSLMode, cfg, "sslMode")
		setStr(&a.TLSRootCert, cfg, "rootCert")
		setStr(&a.TLSCertFile, cfg, "crtText")
		setStr(&a.TLSKeyFile, cfg, "keyText")
	case "ssh":
		// usePassword is derived from key, and password mirrors the key.
		setStr(&a.Username, cfg, "username")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Key, cfg, "sshKey")
	case "kubernetes":
		setStr(&a.ApiServer, cfg, "apiserver")
		setStr(&a.ClusterToken, cfg, "token")
		setStr(&a.ClusterCert, cfg, "cacrt")
		setStr(&a.Namespace, cfg, "namespace")
		setStr(&a.Tolerations, cfg, "tolerationsBytes")
		setStr(&a.Annotations, cfg, "annotationsBytes")
		setStr(&a.NodeSelector, cfg, "nodeSelectorBytes")
		setStr(&a.NodeAffinity, cfg, "affinityBytes")
	case "awsdocumentdb":
		setStr(&a.URI, cfg, "uri")
	case "zerotier":
		setStr(&a.NetworkID, cfg, "network_id")
		setStr(&a.APIToken, cfg, "api_token")
	case "mongodb_atlas":
		setStr(&a.URI, cfg, "uri")
		setStr(&a.OrganizationID, cfg, "organization_id")
		setStr(&a.PublicKey, cfg, "public_key")
		setStr(&a.PrivateKey, cfg, "private_key")
		setStr(&a.ProjectID, cfg, "project_id")
	case "rdp_windows":
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Port, cfg, "port")
	case "awssecretsmanager":
		setStr(&a.AWSRegionName, cfg, "aws_region_name")
		setStr(&a.AWSArn, cfg, "aws_arn")
	case "postgres_aws_secrets_manager", "mysql_aws_secrets_manager",
		"sqlserver_aws_secrets_manager", "snowflake_aws_secrets_manager":
		setStr(&a.Arn, cfg, "arn")
		setStr(&a.Region, cfg, "region")
		setStr(&a.SecretID, cfg, "secret_id")
	case "mongodb_aws_secrets_manager":
		setStr(&a.Arn, cfg, "arn")
		setStr(&a.Region, cfg, "region")
		setStr(&a.SecretID, cfg, "secret_id")
		setStr(&a.Key, cfg, "key")
	case "sql_server":
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
	case "azuresqlserver":
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.DatabaseName, cfg, "databaseName")
	case "splunk":
		setStr(&a.TokenID, cfg, "tokenID")
		setStr(&a.URL, cfg, "url")
	case "datadog":
		setStr(&a.DdSite, cfg, "dd_site")
		setStr(&a.DdApiKey, cfg, "dd_api_key")
	case "coralogix":
		setStr(&a.URI, cfg, "url")
		setStr(&a.PrivateKey, cfg, "privateKey")
		setStr(&a.ApplicationName, cfg, "applicationName")
		setStr(&a.SubSystemName, cfg, "subSystemName")
	case "jumpcloud":
		setStr(&a.ClientID, cfg, "clientID")
		setStr(&a.ClientSecret, cfg, "clientSecret")
		setStr(&a.Domain, cfg, "domain")
		setStr(&a.APIToken, cfg, "apiKey")
	case "msteams":
		setStr(&a.ClientID, cfg, "appID")
		setStr(&a.ClientSecret, cfg, "appKey")
		setStr(&a.TenantID, cfg, "tenantID")
	case "yugabytedb":
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.SSLMode, cfg, "sslMode")
		setStr(&a.RootCert, cfg, "rootCert")
		setStr(&a.Port, cfg, "port")
	case "onelogin":
		setStr(&a.Domain, cfg, "domain")
		setStr(&a.ClientID, cfg, "clientID")
		setStr(&a.ClientSecret, cfg, "clientSecret")
		setStr(&a.ApiClientID, cfg, "apiClientID")
		setStr(&a.ApiClientSecret, cfg, "apiClientSecret")
	case "elasticsearch":
		setStr(&a.URI, cfg, "url")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Index, cfg, "index")
	case "paloalto_ngfw":
		setStr(&a.Password, cfg, "password")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.WebuiPort, cfg, "webui_port")
		setStr(&a.LoginURL, cfg, "login_url")
	case "fortinet_ngfw", "cisco_ngfw", "hpe_switch":
		// `type` is a per-case constant of the forward mapping, not an input.
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.URI, cfg, "login_url")
		setStr(&a.Port, cfg, "port")
		setBool(&a.UseProxy, cfg, "use_proxy")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.WebuiPort, cfg, "webui_port")
	case "snowflake":
		setStr(&a.Hostname, cfg, "databaseAccount")
		setStr(&a.Username, cfg, "databaseUsername")
		setStr(&a.Password, cfg, "databasePassword")
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Warehouse, cfg, "warehouse")
		setStr(&a.Schema, cfg, "schema")
		setStr(&a.Clientcert, cfg, "clientcert")
		setStr(&a.Role, cfg, "role")
	case "custom_siem_webhook":
		setStr(&a.URI, cfg, "url")
		setStr(&a.SharedSecret, cfg, "sharedSecret")
	case "aruba_sw":
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
	case "aruba_instant_on":
		setStr(&a.Host, cfg, "host")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.APIToken, cfg, "apiToken")
	case "syslog":
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Protocol, cfg, "protocol")
	case "customintegration":
		setStr(&a.Image, cfg, "image")
		setStr(&a.ServiceAccountName, cfg, "service_account_name")
	case "clickhouse":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.SSLMode, cfg, "sslMode")
	case "keyspaces":
		setBool(&a.UseServiceAccount, cfg, "use_service_account")
		setBool(&a.CreateIfNotExists, cfg, "create_if_not_exists")
	case "rabbitmq":
		setStr(&a.URI, cfg, "url")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
	case "azurecosmosnosql":
		setStr(&a.URI, cfg, "endpoint")
		setStr(&a.APIToken, cfg, "key")
	case "services":
		setStr(&a.URLs, cfg, "urls")
	case "serverlist":
		setHosts(&a.Hosts, cfg, "hosts")
		setStr(&a.DefaultUser, cfg, "user")
		setStr(&a.Key, cfg, "sshKey")
		setStr(&a.Password, cfg, "password")
	case "msteams_workflow":
		setStr(&a.WebhookURL, cfg, "webhookURL")
	case "1password":
		setStr(&a.ServiceAccount, cfg, "service_account")
		setBool(&a.UseConnectServer, cfg, "use_connect_server")
		setStr(&a.ConnectServerURL, cfg, "connect_server_url")
	case "adaptive_rdp":
		setStr(&a.Targets, cfg, "targets")
	case "adaptiveencrypt":
		setStr(&a.Value, cfg, "value")
	case "adaptiveremotedesktop":
		setStr(&a.CPU, cfg, "cpu")
		setStr(&a.Memory, cfg, "memory")
		setStr(&a.Storage, cfg, "storage")
	case "avastbusinessedr":
		setStr(&a.ClientID, cfg, "clientID")
		setStr(&a.ClientSecret, cfg, "clientSecret")
	case "awscloudwatchlogs":
		setStr(&a.AWSRegionName, cfg, "aws_region")
		setStr(&a.Arn, cfg, "arn")
		setStr(&a.LogGroupName, cfg, "log_group_name")
		setStr(&a.LogStreamName, cfg, "log_stream_name")
	case "awsdocumentdb_aws_secret_manager":
		setStr(&a.Arn, cfg, "arn")
		setStr(&a.Region, cfg, "region")
		setStr(&a.SecretID, cfg, "secret_id")
		setBool(&a.TLSEnabled, cfg, "tlsEnabled")
	case "awselasticcache":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Host, cfg, "host")
		setStr(&a.Port, cfg, "port")
		setBool(&a.TLSEnabled, cfg, "tlsEnabled")
		setBool(&a.TLSSkipVerify, cfg, "tlsSkipVerify")
		setStr(&a.AccessControlMethod, cfg, "access_control_method")
		setStr(&a.AccessControlGroup, cfg, "access_control_group")
		setStr(&a.AWSRegionName, cfg, "aws_region_name")
		setStr(&a.AccessKeyID, cfg, "aws_access_key_id")
		setStr(&a.SecretAccessKey, cfg, "aws_secret_access_key")
	case "azure_documentdb":
		setStr(&a.URI, cfg, "uri")
	case "big_query":
		setStr(&a.CredentialJSON, cfg, "credential_json")
	case "chrome":
		setStr(&a.URL, cfg, "url")
		setStr(&a.Prestart, cfg, "prestart")
		setStr(&a.AutomationMode, cfg, "automationMode")
		setStr(&a.Fields, cfg, "fields")
		setStr(&a.Script, cfg, "script")
	case "cockroachdb_aws_secrets_manager":
		setStr(&a.Arn, cfg, "arn")
		setStr(&a.Region, cfg, "region")
		setStr(&a.SecretID, cfg, "secret_id")
		setStr(&a.Username, cfg, "db_user")
		setStr(&a.Password, cfg, "db_password")
		setStr(&a.DatabaseName, cfg, "db_name")
		setStr(&a.Host, cfg, "db_host")
		setStr(&a.Port, cfg, "db_port")
		setStr(&a.RootCert, cfg, "db_rootCert")
		setStr(&a.SSLMode, cfg, "sslMode")
	case "confluent":
		setStr(&a.OrganizationID, cfg, "organization_id")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Resource, cfg, "resource")
		setStr(&a.APIKey, cfg, "apiKey")
		setStr(&a.APISecret, cfg, "apiSecret")
	case "digitalocean":
		setStr(&a.APIToken, cfg, "api_token")
	case "fortinet_analyzer", "fortinet_manager":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.WebuiPort, cfg, "webui_port")
		setStr(&a.LoginURL, cfg, "login_url")
		setStr(&a.Key, cfg, "sshKey")
	case "heroku":
		setStr(&a.Machine, cfg, "machine")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Token, cfg, "token")
	case "ivanti":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.WebuiPort, cfg, "webui_port")
		setStr(&a.LoginURL, cfg, "login_url")
		setBool(&a.UseProxy, cfg, "use_proxy")
		setStr(&a.Key, cfg, "sshKey")
	case "juniper_sw", "sophos_fw":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Key, cfg, "sshKey")
	case "kafka":
		setStr(&a.ClientConfiguration, cfg, "client_configuration")
	case "ldap":
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.LdapEncryptionMethod, cfg, "ldap-encryption-method")
		setStr(&a.LdapSearchBindDN, cfg, "ldap-search-bind-dn")
		setStr(&a.LdapSearchBindPassword, cfg, "ldap-search-bind-password")
		setStr(&a.LdapUserNameAttribute, cfg, "ldap-user-name-attribute")
		setStr(&a.LdapUserBaseDN, cfg, "ldap-user-base-dn")
	case "mongodb-do":
		setStr(&a.DatabaseName, cfg, "dbname")
		setStr(&a.APIToken, cfg, "api_token")
	case "mongodb36":
		setStr(&a.URI, cfg, "uri")
		setStr(&a.ClientCertificate, cfg, "clientCertificate")
		setBool(&a.UseTLS, cfg, "useTLS")
	case "oracle":
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.ServiceName, cfg, "service_name")
	case "proxysql":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.DatabaseName, cfg, "databaseName")
		setStr(&a.Host, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.SSLMode, cfg, "sslMode")
		setStr(&a.RootCert, cfg, "rootCert")
		setStr(&a.ClientCert, cfg, "clientCert")
		setStr(&a.ClientKey, cfg, "clientKey")
		setBool(&a.OldVersion, cfg, "oldVersion")
		setStr(&a.ProxysqlAdminPort, cfg, "proxysqlAdminPort")
		setStr(&a.ProxysqlAdminUsername, cfg, "proxysqlAdminUsername")
		setStr(&a.ProxysqlAdminPassword, cfg, "proxysqlAdminPassword")
		setStr(&a.ProxysqlHostgroupID, cfg, "proxysqlHostgroupID")
	case "rdpldap":
		setStr(&a.Hostname, cfg, "hostname")
		setStr(&a.Port, cfg, "port")
		setStr(&a.LdapHostname, cfg, "ldap-hostname")
		setStr(&a.LdapPort, cfg, "ldap-port")
		setStr(&a.LdapEncryptionMethod, cfg, "ldap-encryption-method")
		setStr(&a.LdapSearchBindDN, cfg, "ldap-search-bind-dn")
		setStr(&a.LdapSearchBindPassword, cfg, "ldap-search-bind-password")
		setStr(&a.LdapUserBaseDN, cfg, "ldap-user-base-dn")
		setStr(&a.LdapUserNameAttribute, cfg, "ldap-user-name-attribute")
	case "redis":
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Host, cfg, "host")
		setStr(&a.Port, cfg, "port")
		setBool(&a.TLSEnabled, cfg, "tlsEnabled")
		setBool(&a.TLSSkipVerify, cfg, "tlsSkipVerify")
		setBool(&a.IsRedisLabs, cfg, "isRedisLabs")
		setStr(&a.TLSCertFile, cfg, "crtText")
		setStr(&a.TLSKeyFile, cfg, "keyText")
		setStr(&a.TLSCACert, cfg, "caText")
	case "securetunnels":
		// name only, and the server strips it from the configuration blob.
	case "slack_webhook":
		setStr(&a.WebhookURL, cfg, "webhookURL")
		setStr(&a.APIToken, cfg, "apiToken")
	case "vnc":
		setHosts(&a.Hosts, cfg, "hosts")
		setStr(&a.Port, cfg, "port")
		setStr(&a.Password, cfg, "password")
	case "windows-server-groups":
		setHosts(&a.Hosts, cfg, "hosts")
		setStr(&a.Username, cfg, "username")
		setStr(&a.Password, cfg, "password")
		setStr(&a.Port, cfg, "port")
	}
}

// getStr reads a configuration value as a string. Values arrive as decoded JSON,
// so a port written unquoted in the stored YAML comes back as a number.
func getStr(cfg map[string]any, key string) (string, bool) {
	v, ok := cfg[key]
	if !ok {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case int:
		return strconv.Itoa(t), true
	case nil:
		return "", true
	}
	return "", false
}

// getBool reads a configuration value as a bool, accepting the string spellings
// the resource forms submit for checkboxes.
func getBool(cfg map[string]any, key string) (bool, bool) {
	v, ok := cfg[key]
	if !ok {
		return false, false
	}
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		b, err := strconv.ParseBool(t)
		if err != nil {
			return false, false
		}
		return b, true
	}
	return false, false
}

// argProbeSentinel is a value no real configuration can hold, used to trace a
// server config key back to the argument it feeds.
const argProbeSentinel = "\x00adaptive-arg-probe\x00"

// argForConfigKey resolves the schema property that a server config key feeds.
//
// The mapping lives in applyIntegrationConfig's switch, one arm per integration
// type. It is queried here rather than duplicated into a table: a parallel table
// would be ~82 types long and would silently rot the first time someone edited
// only one of the two. Running the real switch with a sentinel and seeing where
// it lands cannot disagree with itself.
//
// ResourceArgs is a flat struct of *string fields carrying their own pulumi
// tags, and setStr writes any non-empty value unconditionally, so the sentinel
// always lands on exactly the field the key drives. Non-pointer fields (tags,
// the setHosts targets) are skipped: no secret is carried in one.
//
// Reports ok=false for a key the type does not map, and for the dotted or
// indexed paths the server produces for nested secrets ("targets[0].password") —
// those name no single argument, so the caller reports them by name instead.
func argForConfigKey(integrationType, cfgKey string) (field int, prop string, ok bool) {
	if cfgKey == "" {
		return 0, "", false
	}
	var probe ResourceArgs
	applyIntegrationConfig(&probe, integrationType, map[string]any{cfgKey: argProbeSentinel})

	v := reflect.ValueOf(probe)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() != reflect.Ptr || f.IsNil() || f.Elem().Kind() != reflect.String {
			continue
		}
		if f.Elem().String() != argProbeSentinel {
			continue
		}
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("pulumi"), ",")
		if name == "" {
			return 0, "", false
		}
		return i, name, true
	}
	return 0, "", false
}

// argIsSecret reports whether the field at index i of ResourceArgs is declared
// secret, so a key the server redacted is only ever marked on an argument the
// schema also treats as secret.
func argIsSecret(field int) bool {
	return reflect.TypeOf(ResourceArgs{}).Field(field).Tag.Get("provider") == "secret"
}

// argIsSet reports whether the program supplied a value for the field at index
// i. An unset argument means the secret is not under management: the drift is
// still reported, and the next update clears it server-side, because the
// program is the source of truth for the whole configuration.
func argIsSet(a *ResourceArgs, field int) bool {
	f := reflect.ValueOf(a).Elem().Field(field)
	return f.Kind() == reflect.Ptr && !f.IsNil()
}

// setArg writes a value to the field at index i.
func setArg(a *ResourceArgs, field int, value string) {
	f := reflect.ValueOf(a).Elem().Field(field)
	if f.Kind() != reflect.Ptr || f.Type().Elem().Kind() != reflect.String {
		return
	}
	v := value
	f.Set(reflect.ValueOf(&v))
}

// setStr reconciles one optional string argument against the server value. An
// absent key (a stripped secret) leaves the argument alone; an empty value
// clears it only when the program had set it.
func setStr(dst **string, cfg map[string]any, key string) {
	s, ok := getStr(cfg, key)
	if !ok {
		return
	}
	if s == "" {
		if *dst != nil {
			*dst = nil
		}
		return
	}
	v := s
	*dst = &v
}

// setBool reconciles one optional bool argument. A server-side false is never
// adopted into an argument the program left unset: several of these flags
// default to false server-side, and writing them back would show as a diff.
func setBool(dst **bool, cfg map[string]any, key string) {
	b, ok := getBool(cfg, key)
	if !ok {
		return
	}
	if !b && *dst == nil {
		return
	}
	v := b
	*dst = &v
}

// setHosts splits a newline-joined host list back into the slice the program
// wrote. Ordering is preserved, so a list that did not drift compares equal.
func setHosts(dst *[]string, cfg map[string]any, key string) {
	s, ok := getStr(cfg, key)
	if !ok {
		return
	}
	var hosts []string
	for _, h := range strings.Split(s, "\n") {
		if h = strings.TrimSpace(h); h != "" {
			hosts = append(hosts, h)
		}
	}
	if len(hosts) == 0 && len(*dst) == 0 {
		return
	}
	*dst = hosts
}
