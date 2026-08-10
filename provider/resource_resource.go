package adaptive

import (
	"context"
	"fmt"

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
	Password     *string `pulumi:"password,optional" provider:"secret"`
	DatabaseName *string `pulumi:"databaseName,optional"`
	SSLMode      *string `pulumi:"sslMode,optional"`
	Protocol     *string `pulumi:"protocol,optional"`
	URL          *string `pulumi:"url,optional"`

	// RDS IAM authentication (postgres). Supported by the Terraform provider
	// since the RDS IAM release; carried here so both providers describe the
	// same resource.
	UseRdsIam          *bool   `pulumi:"useRdsIam,optional"`
	UseIrsa            *bool   `pulumi:"useIrsa,optional"`
	AWSRegion          *string `pulumi:"awsRegion,optional"`
	AWSRoleArn         *string `pulumi:"awsRoleArn,optional"`
	AWSAccessKeyID     *string `pulumi:"awsAccessKeyId,optional"`
	AWSSecretAccessKey *string `pulumi:"awsSecretAccessKey,optional" provider:"secret"`
	AWSServiceAccount  *string `pulumi:"awsServiceAccount,optional"`

	// TLS / certs
	RootCert    *string `pulumi:"rootCert,optional" provider:"secret"`
	TLSRootCert *string `pulumi:"tlsRootCert,optional" provider:"secret"`
	TLSCertFile *string `pulumi:"tlsCertFile,optional" provider:"secret"`
	TLSKeyFile  *string `pulumi:"tlsKeyFile,optional" provider:"secret"`

	// Kubernetes
	ApiServer    *string `pulumi:"apiServer,optional"`
	ClusterToken *string `pulumi:"clusterToken,optional" provider:"secret"`
	ClusterCert  *string `pulumi:"clusterCert,optional" provider:"secret"`
	Namespace    *string `pulumi:"namespace,optional"`
	Tolerations  *string `pulumi:"tolerations,optional"`
	Annotations  *string `pulumi:"annotations,optional"`
	NodeSelector *string `pulumi:"nodeSelector,optional"`
	NodeAffinity *string `pulumi:"nodeAffinity,optional"`

	// AWS
	RegionName      *string `pulumi:"regionName,optional"`
	AccessKeyID     *string `pulumi:"accessKeyId,optional" provider:"secret"`
	SecretAccessKey *string `pulumi:"secretAccessKey,optional" provider:"secret"`
	Arn             *string `pulumi:"arn,optional"`
	Region          *string `pulumi:"region,optional"`
	SecretID        *string `pulumi:"secretId,optional"`
	AWSArn          *string `pulumi:"awsArn,optional"`
	AWSRegionName   *string `pulumi:"awsRegionName,optional"`

	// Azure
	TenantID        *string `pulumi:"tenantId,optional"`
	ApplicationID   *string `pulumi:"applicationId,optional"`
	ClientSecret    *string `pulumi:"clientSecret,optional" provider:"secret"`
	ApiClientID     *string `pulumi:"apiClientId,optional"`
	ApiClientSecret *string `pulumi:"apiClientSecret,optional" provider:"secret"`
	UseTenant       *bool   `pulumi:"useTenant,optional"`

	// GCP / Google / OAuth identity
	ProjectID *string `pulumi:"projectId,optional"`
	KeyFile   *string `pulumi:"keyFile,optional" provider:"secret"`
	Domain    *string `pulumi:"domain,optional"`
	ClientID  *string `pulumi:"clientId,optional"`
	LoginURL  *string `pulumi:"loginUrl,optional"`

	// Snowflake
	Warehouse  *string `pulumi:"warehouse,optional"`
	Schema     *string `pulumi:"schema,optional"`
	Clientcert *string `pulumi:"clientcert,optional" provider:"secret"`
	Role       *string `pulumi:"role,optional"`

	// Services / serverlist
	URLs        *string  `pulumi:"urls,optional"`
	Hosts       []string `pulumi:"hosts,optional"`
	DefaultUser *string  `pulumi:"defaultUser,optional"`

	// SSH
	Key            *string `pulumi:"key,optional" provider:"secret"`
	PublicKey      *string `pulumi:"publicKey,optional"`
	OrganizationID *string `pulumi:"organizationId,optional"`

	// Misc service fields
	APIToken           *string `pulumi:"apiToken,optional" provider:"secret"`
	PrivateKey         *string `pulumi:"privateKey,optional" provider:"secret"`
	ApplicationName    *string `pulumi:"applicationName,optional"`
	SubSystemName      *string `pulumi:"subSystemName,optional"`
	SharedSecret       *string `pulumi:"sharedSecret,optional" provider:"secret"`
	Image              *string `pulumi:"image,optional"`
	ServiceAccountName *string `pulumi:"serviceAccountName,optional"`
	DdSite             *string `pulumi:"ddSite,optional"`
	DdApiKey           *string `pulumi:"ddApiKey,optional" provider:"secret"`
	Index              *string `pulumi:"index,optional"`
	UseProxy           *bool   `pulumi:"useProxy,optional"`
	WebuiPort          *string `pulumi:"webuiPort,optional"`
	UseServiceAccount  *bool   `pulumi:"useServiceAccount,optional"`
	CreateIfNotExists  *bool   `pulumi:"createIfNotExists,optional"`
	NetworkID          *string `pulumi:"networkId,optional"`
	TokenID            *string `pulumi:"tokenId,optional"`
	APIKey             *string `pulumi:"apiKey,optional" provider:"secret"`
	AppID              *string `pulumi:"appId,optional"`
	AppKey             *string `pulumi:"appKey,optional" provider:"secret"`
	Version            *string `pulumi:"version,optional"`
	DatabaseAccount    *string `pulumi:"databaseAccount,optional"`
	DatabaseUsername   *string `pulumi:"databaseUsername,optional"`
	DatabasePassword   *string `pulumi:"databasePassword,optional" provider:"secret"`
	WebhookURL         *string `pulumi:"webhookUrl,optional" provider:"secret"`
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

func (*Resource) Delete(ctx context.Context, req infer.DeleteRequest[ResourceState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteResource(ctx, req.ID, req.State.Name)
}
