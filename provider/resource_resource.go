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

	// AWS RDS IAM authentication (postgres)
	UseRDSIAM          *bool   `pulumi:"useRdsIam,optional"`
	UseIRSA            *bool   `pulumi:"useIrsa,optional"`
	AWSRegion          *string `pulumi:"awsRegion,optional"`
	AWSRoleARN         *string `pulumi:"awsRoleArn,optional"`
	AWSAccessKeyID     *string `pulumi:"awsAccessKeyId,optional"`
	AWSSecretAccessKey *string `pulumi:"awsSecretAccessKey,optional"`
	AWSServiceAccount  *string `pulumi:"awsServiceAccount,optional"`

	// Azure
	TenantID        *string `pulumi:"tenantId,optional"`
	ApplicationID   *string `pulumi:"applicationId,optional"`
	ClientSecret    *string `pulumi:"clientSecret,optional"`
	ApiClientID     *string `pulumi:"apiClientId,optional"`
	ApiClientSecret *string `pulumi:"apiClientSecret,optional"`
	UseTenant       *bool   `pulumi:"useTenant,optional"`

	// GCP / Google / OAuth identity
	ProjectID    *string `pulumi:"projectId,optional"`
	KeyFile      *string `pulumi:"keyFile,optional"`
	Domain       *string `pulumi:"domain,optional"`
	ClientID     *string `pulumi:"clientId,optional"`
	LoginURL     *string `pulumi:"loginUrl,optional"`

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
}

type ResourceState struct {
	ResourceArgs
}

func (r *ResourceArgs) Annotate(a infer.Annotator) {
	a.Describe(&r.Type, "Type of the Adaptive resource (integration), e.g. postgres, mysql, mongodb, ssh, kubernetes, aws, gcp, azure, snowflake.")
	a.Describe(&r.Name, "Name of the Adaptive resource.")
	a.Describe(&r.Tags, "Optional tags.")
	a.Describe(&r.DefaultCluster, "The default cluster the resource is deployed to.")
	a.Describe(&r.UseRDSIAM, "Postgres only. Authenticate with short-lived AWS RDS IAM auth tokens instead of a stored password. The database user must be granted rds_iam, the RDS instance must have IAM authentication enabled, and sslMode cannot be \"disable\".")
	a.Describe(&r.UseIRSA, "Postgres only. Mint RDS IAM tokens with awsServiceAccount, or with the platform's own IRSA / instance-profile identity when it is empty. Set to false to use awsAccessKeyId / awsSecretAccessKey instead. Leave unset to infer it from whether static keys were given.")
	a.Describe(&r.AWSRegion, "Postgres only. The AWS region used to mint RDS IAM tokens. Leave empty to auto-detect from an *.rds.amazonaws.com hostname.")
	a.Describe(&r.AWSRoleARN, "Postgres only. An IAM role to assume when minting RDS IAM tokens. With IRSA it overrides the service account's eks.amazonaws.com/role-arn annotation.")
	a.Describe(&r.AWSAccessKeyID, "Postgres only. Static AWS access key id used to mint RDS IAM tokens. Requires useIrsa = false.")
	a.Describe(&r.AWSSecretAccessKey, "Postgres only. Static AWS secret access key used to mint RDS IAM tokens. Requires useIrsa = false.")
	a.Describe(&r.AWSServiceAccount, "Postgres only. Name of a Kubernetes ServiceAccount in the session cluster annotated with eks.amazonaws.com/role-arn; the platform mints RDS IAM tokens on its behalf. Requires useIrsa = true.")
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
