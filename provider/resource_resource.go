package adaptive

import (
	"context"
	"fmt"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"gopkg.in/yaml.v2"
)

// Resource (adaptive:index:Resource) is the general-purpose integration
// resource. The `type` field selects the integration (postgres, mysql, ssh,
// kubernetes, aws, ...) and the remaining optional fields are that
// integration's settings, mirroring the Terraform `adaptive_resource`.
type Resource struct{}

type ResourceArgs struct {
	// Core
	Type           string   `pulumi:"type"`
	Name           string   `pulumi:"name"`
	Tags           []string `pulumi:"tags,optional"`
	DefaultCluster *string  `pulumi:"defaultCluster,optional"`

	// Common connection fields
	URI          *string `pulumi:"uri,optional"`
	Host         *string `pulumi:"host,optional"`
	Hostname     *string `pulumi:"hostname,optional"`
	Port         *string `pulumi:"port,optional"`
	Username     *string `pulumi:"username,optional"`
	Password     *string `pulumi:"password,optional"`
	DatabaseName *string `pulumi:"databaseName,optional"`
	SSLMode      *string `pulumi:"sslMode,optional"`
	Protocol     *string `pulumi:"protocol,optional"`
	URL          *string `pulumi:"url,optional"`

	// TLS / certs
	RootCert    *string `pulumi:"rootCert,optional"`
	TLSRootCert *string `pulumi:"tlsRootCert,optional"`
	TLSCertFile *string `pulumi:"tlsCertFile,optional"`
	TLSKeyFile  *string `pulumi:"tlsKeyFile,optional"`

	// Kubernetes
	ApiServer    *string `pulumi:"apiServer,optional"`
	ClusterToken *string `pulumi:"clusterToken,optional"`
	ClusterCert  *string `pulumi:"clusterCert,optional"`
	Namespace    *string `pulumi:"namespace,optional"`
	Tolerations  *string `pulumi:"tolerations,optional"`
	Annotations  *string `pulumi:"annotations,optional"`
	NodeSelector *string `pulumi:"nodeSelector,optional"`
	NodeAffinity *string `pulumi:"nodeAffinity,optional"`

	// AWS
	RegionName      *string `pulumi:"regionName,optional"`
	AccessKeyID     *string `pulumi:"accessKeyId,optional"`
	SecretAccessKey *string `pulumi:"secretAccessKey,optional"`
	Arn             *string `pulumi:"arn,optional"`
	Region          *string `pulumi:"region,optional"`
	SecretID        *string `pulumi:"secretId,optional"`
	AWSArn          *string `pulumi:"awsArn,optional"`
	AWSRegionName   *string `pulumi:"awsRegionName,optional"`

	// Azure
	TenantID        *string `pulumi:"tenantId,optional"`
	ApplicationID   *string `pulumi:"applicationId,optional"`
	ClientSecret    *string `pulumi:"clientSecret,optional"`
	ApiClientID     *string `pulumi:"apiClientId,optional"`
	ApiClientSecret *string `pulumi:"apiClientSecret,optional"`
	UseTenant       *bool   `pulumi:"useTenant,optional"`

	// GCP / Google / OAuth identity
	ProjectID *string `pulumi:"projectId,optional"`
	KeyFile   *string `pulumi:"keyFile,optional"`
	Domain    *string `pulumi:"domain,optional"`
	ClientID  *string `pulumi:"clientId,optional"`
	LoginURL  *string `pulumi:"loginUrl,optional"`

	// Snowflake
	Warehouse  *string `pulumi:"warehouse,optional"`
	Schema     *string `pulumi:"schema,optional"`
	Clientcert *string `pulumi:"clientcert,optional"`
	Role       *string `pulumi:"role,optional"`

	// Services / serverlist
	URLs        *string  `pulumi:"urls,optional"`
	Hosts       []string `pulumi:"hosts,optional"`
	DefaultUser *string  `pulumi:"defaultUser,optional"`

	// SSH
	Key            *string `pulumi:"key,optional"`
	PublicKey      *string `pulumi:"publicKey,optional"`
	OrganizationID *string `pulumi:"organizationId,optional"`

	// Misc service fields
	APIToken           *string `pulumi:"apiToken,optional"`
	PrivateKey         *string `pulumi:"privateKey,optional"`
	ApplicationName    *string `pulumi:"applicationName,optional"`
	SubSystemName      *string `pulumi:"subSystemName,optional"`
	SharedSecret       *string `pulumi:"sharedSecret,optional"`
	Image              *string `pulumi:"image,optional"`
	ServiceAccountName *string `pulumi:"serviceAccountName,optional"`
	DdSite             *string `pulumi:"ddSite,optional"`
	DdApiKey           *string `pulumi:"ddApiKey,optional"`
	Index              *string `pulumi:"index,optional"`
	UseProxy           *bool   `pulumi:"useProxy,optional"`
	WebuiPort          *string `pulumi:"webuiPort,optional"`
	UseServiceAccount  *bool   `pulumi:"useServiceAccount,optional"`
	CreateIfNotExists  *bool   `pulumi:"createIfNotExists,optional"`
	NetworkID          *string `pulumi:"networkId,optional"`
	TokenID            *string `pulumi:"tokenId,optional"`
	APIKey             *string `pulumi:"apiKey,optional"`
	AppID              *string `pulumi:"appId,optional"`
	AppKey             *string `pulumi:"appKey,optional"`
	Version            *string `pulumi:"version,optional"`
	DatabaseAccount    *string `pulumi:"databaseAccount,optional"`
	DatabaseUsername   *string `pulumi:"databaseUsername,optional"`
	DatabasePassword   *string `pulumi:"databasePassword,optional"`
	WebhookURL         *string `pulumi:"webhookUrl,optional"`

	// TLS toggles (redis, elasticache, documentdb secrets manager, mongodb36)
	TLSEnabled    *bool   `pulumi:"tlsEnabled,optional"`
	TLSSkipVerify *bool   `pulumi:"tlsSkipVerify,optional"`
	UseTLS        *bool   `pulumi:"useTls,optional"`
	TLSCACert     *string `pulumi:"tlsCaCert,optional"`
	ClientCert    *string `pulumi:"clientCert,optional"`
	ClientKey     *string `pulumi:"clientKey,optional"`

	// LDAP (ldap, rdpldap)
	LdapHostname           *string `pulumi:"ldapHostname,optional"`
	LdapPort               *string `pulumi:"ldapPort,optional"`
	LdapEncryptionMethod   *string `pulumi:"ldapEncryptionMethod,optional"`
	LdapSearchBindDN       *string `pulumi:"ldapSearchBindDn,optional"`
	LdapSearchBindPassword *string `pulumi:"ldapSearchBindPassword,optional"`
	LdapUserNameAttribute  *string `pulumi:"ldapUserNameAttribute,optional"`
	LdapUserBaseDN         *string `pulumi:"ldapUserBaseDn,optional"`

	// ProxySQL admin interface
	OldVersion            *bool   `pulumi:"oldVersion,optional"`
	ProxysqlAdminPort     *string `pulumi:"proxysqlAdminPort,optional"`
	ProxysqlAdminUsername *string `pulumi:"proxysqlAdminUsername,optional"`
	ProxysqlAdminPassword *string `pulumi:"proxysqlAdminPassword,optional"`
	ProxysqlHostgroupID   *string `pulumi:"proxysqlHostgroupId,optional"`

	// Chrome automation
	AutomationMode *string `pulumi:"automationMode,optional"`
	Fields         *string `pulumi:"fields,optional"`
	Script         *string `pulumi:"script,optional"`
	Prestart       *string `pulumi:"prestart,optional"`

	// Remote desktop sizing (adaptiveremotedesktop)
	CPU     *string `pulumi:"cpu,optional"`
	Memory  *string `pulumi:"memory,optional"`
	Storage *string `pulumi:"storage,optional"`

	// Remaining per-integration fields
	ServiceAccount      *string `pulumi:"serviceAccount,optional"`
	UseConnectServer    *bool   `pulumi:"useConnectServer,optional"`
	ConnectServerURL    *string `pulumi:"connectServerUrl,optional"`
	Targets             *string `pulumi:"targets,optional"`
	Value               *string `pulumi:"value,optional"`
	LogGroupName        *string `pulumi:"logGroupName,optional"`
	LogStreamName       *string `pulumi:"logStreamName,optional"`
	AccessControlMethod *string `pulumi:"accessControlMethod,optional"`
	AccessControlGroup  *string `pulumi:"accessControlGroup,optional"`
	CredentialJSON      *string `pulumi:"credentialJson,optional"`
	Resource            *string `pulumi:"resource,optional"`
	APISecret           *string `pulumi:"apiSecret,optional"`
	Machine             *string `pulumi:"machine,optional"`
	Token               *string `pulumi:"token,optional"`
	ClientConfiguration *string `pulumi:"clientConfiguration,optional"`
	ClientCertificate   *string `pulumi:"clientCertificate,optional"`
	ServiceName         *string `pulumi:"serviceName,optional"`
	IsRedisLabs         *bool   `pulumi:"isRedisLabs,optional"`
}

type ResourceState struct {
	ResourceArgs
}

func (r *ResourceArgs) Annotate(a infer.Annotator) {
	a.Describe(&r.Type, "Type of the Adaptive resource (integration), e.g. postgres, mysql, mongodb, ssh, kubernetes, aws, gcp, azure, snowflake.")
	a.Describe(&r.Name, "Name of the Adaptive resource.")
	a.Describe(&r.Tags, "Optional tags.")
	a.Describe(&r.DefaultCluster, "The default cluster the resource is deployed to.")
}

func (*Resource) Create(ctx context.Context, req infer.CreateRequest[ResourceArgs]) (infer.CreateResponse[ResourceState], error) {
	out := infer.CreateResponse[ResourceState]{Output: ResourceState{ResourceArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	cfgObj, effType, err := buildIntegrationConfig(req.Inputs)
	if err != nil {
		return out, err
	}
	yamlBytes, err := yaml.Marshal(cfgObj)
	if err != nil {
		return out, fmt.Errorf("could not marshal resource configuration: %w", err)
	}
	resp, err := c.CreateResource(ctx, req.Inputs.Name, effType, yamlBytes, req.Inputs.Tags, sv(req.Inputs.DefaultCluster))
	if err != nil {
		return out, err
	}
	out.ID = resp.ID
	return out, nil
}

func (*Resource) Update(ctx context.Context, req infer.UpdateRequest[ResourceArgs, ResourceState]) (infer.UpdateResponse[ResourceState], error) {
	out := infer.UpdateResponse[ResourceState]{Output: ResourceState{ResourceArgs: req.Inputs}}
	if req.DryRun {
		return out, nil
	}
	c, err := clientFromConfig(ctx)
	if err != nil {
		return out, err
	}
	cfgObj, effType, err := buildIntegrationConfig(req.Inputs)
	if err != nil {
		return out, err
	}
	yamlBytes, err := yaml.Marshal(cfgObj)
	if err != nil {
		return out, fmt.Errorf("could not marshal resource configuration: %w", err)
	}
	_, err = c.UpdateResource(ctx, req.ID, effType, yamlBytes, req.Inputs.Tags, sv(req.Inputs.DefaultCluster))
	return out, err
}

func (*Resource) Read(ctx context.Context, req infer.ReadRequest[ResourceArgs, ResourceState]) (infer.ReadResponse[ResourceArgs, ResourceState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, err
	}
	r, err := c.ReadResource(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, err
	}
	if r == nil {
		// Deleted out-of-band: an empty response drops the resource from state.
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, nil
	}

	// On import there are no prior inputs to reconcile against, so everything is
	// taken from the server. On refresh we start from what the program wrote and
	// overwrite only what the server actually reports, which is what keeps the
	// credentials it withholds in state.
	isImport := req.Inputs.Name == "" && req.Inputs.Type == ""

	inputs := req.Inputs
	if isImport {
		inputs = ResourceArgs{}
	}
	inputs.Name = r.Name
	inputs.Type = providerType(r.IntegrationType)
	inputs.Tags = setList(inputs.Tags, r.UserTags)
	inputs.DefaultCluster = strOpt(inputs.DefaultCluster, r.DefaultCluster, isImport)
	applyIntegrationConfig(&inputs, r.IntegrationType, r.Configuration)

	if isImport && len(r.RedactedKeys) > 0 {
		p.GetLogger(ctx).Warningf(
			"resource %q was imported without its credentials: the server withholds %s. "+
				"Set the matching arguments in your program, or the next update will clear them.",
			r.Name, strings.Join(r.RedactedKeys, ", "))
	}

	return infer.ReadResponse[ResourceArgs, ResourceState]{
		ID:     req.ID,
		Inputs: inputs,
		State:  ResourceState{ResourceArgs: inputs},
	}, nil
}

func (*Resource) Delete(ctx context.Context, req infer.DeleteRequest[ResourceState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteResource(ctx, req.ID, req.State.Name)
}
