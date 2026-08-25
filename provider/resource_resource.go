package adaptive

import (
	"context"
	"fmt"
	"sort"
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
	URI          *string `pulumi:"uri,optional" provider:"secret"`
	Host         *string `pulumi:"host,optional"`
	Hostname     *string `pulumi:"hostname,optional"`
	Port         *string `pulumi:"port,optional"`
	Username     *string `pulumi:"username,optional"`
	Password     *string `pulumi:"password,optional" provider:"secret"`
	DatabaseName *string `pulumi:"databaseName,optional"`
	SSLMode      *string `pulumi:"sslMode,optional"`
	Protocol     *string `pulumi:"protocol,optional"`
	URL          *string `pulumi:"url,optional"`

	// TLS / certs
	RootCert    *string `pulumi:"rootCert,optional"`
	TLSRootCert *string `pulumi:"tlsRootCert,optional"`
	TLSCertFile *string `pulumi:"tlsCertFile,optional"`
	TLSKeyFile  *string `pulumi:"tlsKeyFile,optional" provider:"secret"`

	// Kubernetes
	ApiServer    *string `pulumi:"apiServer,optional"`
	ClusterToken *string `pulumi:"clusterToken,optional" provider:"secret"`
	ClusterCert  *string `pulumi:"clusterCert,optional"`
	Namespace    *string `pulumi:"namespace,optional"`
	Tolerations  *string `pulumi:"tolerations,optional"`
	Annotations  *string `pulumi:"annotations,optional"`
	NodeSelector *string `pulumi:"nodeSelector,optional"`
	NodeAffinity *string `pulumi:"nodeAffinity,optional"`

	// AWS
	RegionName      *string `pulumi:"regionName,optional"`
	AccessKeyID     *string `pulumi:"accessKeyId,optional"`
	SecretAccessKey *string `pulumi:"secretAccessKey,optional" provider:"secret"`
	Arn             *string `pulumi:"arn,optional"`
	Region          *string `pulumi:"region,optional"`
	SecretID        *string `pulumi:"secretId,optional" provider:"secret"`
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
	TokenID            *string `pulumi:"tokenId,optional" provider:"secret"`
	APIKey             *string `pulumi:"apiKey,optional" provider:"secret"`
	AppID              *string `pulumi:"appId,optional"`
	AppKey             *string `pulumi:"appKey,optional" provider:"secret"`
	Version            *string `pulumi:"version,optional"`
	DatabaseAccount    *string `pulumi:"databaseAccount,optional"`
	DatabaseUsername   *string `pulumi:"databaseUsername,optional"`
	DatabasePassword   *string `pulumi:"databasePassword,optional" provider:"secret"`
	WebhookURL         *string `pulumi:"webhookUrl,optional" provider:"secret"`

	// TLS toggles (redis, elasticache, documentdb secrets manager, mongodb36)
	TLSEnabled    *bool   `pulumi:"tlsEnabled,optional"`
	TLSSkipVerify *bool   `pulumi:"tlsSkipVerify,optional"`
	UseTLS        *bool   `pulumi:"useTls,optional"`
	TLSCACert     *string `pulumi:"tlsCaCert,optional"`
	ClientCert    *string `pulumi:"clientCert,optional" provider:"secret"`
	ClientKey     *string `pulumi:"clientKey,optional" provider:"secret"`

	// LDAP (ldap, rdpldap)
	LdapHostname           *string `pulumi:"ldapHostname,optional"`
	LdapPort               *string `pulumi:"ldapPort,optional"`
	LdapEncryptionMethod   *string `pulumi:"ldapEncryptionMethod,optional"`
	LdapSearchBindDN       *string `pulumi:"ldapSearchBindDn,optional"`
	LdapSearchBindPassword *string `pulumi:"ldapSearchBindPassword,optional" provider:"secret"`
	LdapUserNameAttribute  *string `pulumi:"ldapUserNameAttribute,optional"`
	LdapUserBaseDN         *string `pulumi:"ldapUserBaseDn,optional"`

	// ProxySQL admin interface
	OldVersion            *bool   `pulumi:"oldVersion,optional"`
	ProxysqlAdminPort     *string `pulumi:"proxysqlAdminPort,optional"`
	ProxysqlAdminUsername *string `pulumi:"proxysqlAdminUsername,optional"`
	ProxysqlAdminPassword *string `pulumi:"proxysqlAdminPassword,optional" provider:"secret"`
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
	Targets             *string `pulumi:"targets,optional" provider:"secret"`
	Value               *string `pulumi:"value,optional"`
	LogGroupName        *string `pulumi:"logGroupName,optional"`
	LogStreamName       *string `pulumi:"logStreamName,optional"`
	AccessControlMethod *string `pulumi:"accessControlMethod,optional"`
	AccessControlGroup  *string `pulumi:"accessControlGroup,optional"`
	CredentialJSON      *string `pulumi:"credentialJson,optional" provider:"secret"`
	Resource            *string `pulumi:"resource,optional"`
	APISecret           *string `pulumi:"apiSecret,optional" provider:"secret"`
	Machine             *string `pulumi:"machine,optional"`
	Token               *string `pulumi:"token,optional" provider:"secret"`
	ClientConfiguration *string `pulumi:"clientConfiguration,optional" provider:"secret"`
	ClientCertificate   *string `pulumi:"clientCertificate,optional" provider:"secret"`
	ServiceName         *string `pulumi:"serviceName,optional"`
	IsRedisLabs         *bool   `pulumi:"isRedisLabs,optional"`
}

type ResourceState struct {
	ResourceArgs
	// AppliedDigests are the server's secret fingerprints as of our last
	// successful write — deliberately NOT refreshed by Read. Refresh compares
	// the live fingerprints against these; overwriting them on refresh (as the
	// old redactedDigests field did) destroys the very evidence the comparison
	// produced, which is why out-of-band secret changes never reached preview.
	AppliedDigests map[string]string `pulumi:"appliedDigests,optional"`
}

func (r *ResourceArgs) Annotate(a infer.Annotator) {
	a.Describe(&r.Type, "Type of the Adaptive resource (integration), e.g. postgres, mysql, mongodb, ssh, kubernetes, aws, gcp, azure, snowflake.")
	a.Describe(&r.Name, "Name of the Adaptive resource.")
	a.Describe(&r.Tags, "Optional tags.")
	a.Describe(&r.DefaultCluster, "The default cluster the resource is deployed to.")
}

func (r *ResourceState) Annotate(a infer.Annotator) {
	a.Describe(&r.AppliedDigests, "Opaque server fingerprints of the write-only secret fields as of the last "+
		"write by this provider, used to detect out-of-band secret changes on refresh. Not comparable "+
		"across resources or workspaces.")
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
	// Best-effort: capture the server's secret fingerprints for drift detection.
	// When this read fails the fingerprints stay unset, so the next refresh
	// reports our own write as drift. It self-corrects on the next successful
	// write and only ever over-reports, which is the safe direction.
	if r, rerr := c.ReadResource(ctx, resp.ID); rerr == nil && r != nil {
		out.Output.AppliedDigests = r.RedactedDigests
	}
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
	if _, err := c.UpdateResource(ctx, req.ID, effType, yamlBytes, req.Inputs.Tags, sv(req.Inputs.DefaultCluster)); err != nil {
		return out, err
	}
	// Best-effort: re-record the secret fingerprints after the write, so the
	// drift this update just reconciled stops being reported. A failed read
	// leaves the previous fingerprints in place and the next refresh re-reports
	// the same drift — see the note in Create.
	out.Output.AppliedDigests = req.State.AppliedDigests
	if r, rerr := c.ReadResource(ctx, req.ID); rerr == nil && r != nil {
		out.Output.AppliedDigests = r.RedactedDigests
	}
	return out, nil
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
	// On import there are no prior inputs to reconcile against, so everything is
	// taken from the server. On refresh we start from what the program wrote and
	// overwrite only what the server actually reports, which is what keeps the
	// credentials it withholds in state.
	isImport := req.Inputs.Name == "" && req.Inputs.Type == ""

	if r == nil {
		if isImport {
			return infer.ReadResponse[ResourceArgs, ResourceState]{}, notFoundOnImport("resource", req.ID)
		}
		// Deleted out-of-band: an empty response drops the resource from state.
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, nil
	}
	// A 200 carrying no identity is not a real record — a server that answers
	// unknown ids with a zero-valued body would otherwise fabricate a resource
	// out of nothing. Only enforced on import, where there is no prior state to
	// lose.
	if isImport && r.Name == "" {
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, notFoundOnImport("resource", req.ID)
	}

	inputs := req.Inputs
	if isImport {
		inputs = ResourceArgs{}
	}
	inputs.Name = r.Name
	inputs.Type = providerType(r.IntegrationType)
	inputs.Tags = setList(inputs.Tags, r.UserTags)
	inputs.DefaultCluster = strOpt(inputs.DefaultCluster, r.DefaultCluster, isImport)

	applyIntegrationConfig(&inputs, r.IntegrationType, r.Configuration)

	// The applied fingerprints are recorded at write time and deliberately
	// survive a refresh: they are the baseline the live fingerprints are
	// compared against. The one exception is seeding — with nothing recorded
	// there is no baseline, so adopt the current view and report nothing.
	appliedDigests := req.State.AppliedDigests
	if len(appliedDigests) == 0 {
		appliedDigests = r.RedactedDigests
	}

	// Secret drift: the server withholds secret values but returns an opaque
	// fingerprint per redacted key. A fingerprint that differs from the one
	// recorded at our last write means the value changed out-of-band.
	//
	// The signal is the fingerprint written onto the argument it belongs to.
	// Marking the argument rather than blanking it is what makes the drift
	// reach preview at all: blanking only did anything when the program already
	// set the field, so a secret the program does not manage drifted silently.
	// The argument is secret-typed, so the marker renders as [secret] and no
	// value is disclosed — the property name carries the information.
	//
	// What the marker produces:
	//   - program sets the field  -> ~ password, and up re-applies its value
	//   - program does not        -> - password, and up clears it server-side,
	//                                because the program owns the whole config
	//
	// Nothing writes state inputs back to the server (Create and Update build
	// from req.Inputs, the program's inputs), so a fingerprint sitting in a
	// secret argument can never be applied as one.
	//
	// An empty recorded map (import, a resource predating this field, an older
	// server) is seeded rather than reported: everything would look drifted.
	if !isImport && len(r.RedactedDigests) > 0 && len(req.State.AppliedDigests) > 0 {
		var unresolved []string
		for _, k := range driftedDigestKeys(req.State.AppliedDigests, r.RedactedDigests) {
			field, _, ok := argForConfigKey(r.IntegrationType, k)
			if !ok || !argIsSecret(field) {
				unresolved = append(unresolved, k)
				continue
			}
			setArg(&inputs, field, r.RedactedDigests[k])
		}
		// Some keys name no single argument: the dotted and indexed paths the
		// server emits for nested secrets, and the handful of config keys that
		// mirror another (ssh writes the key material to both sshKey and
		// password, and the read arm reconciles via sshKey). Reporting them by
		// name keeps the drift visible rather than dropping it on the floor.
		if len(unresolved) > 0 {
			p.GetLogger(ctx).Warningf(
				"resource %q: the secret behind %s changed outside this program, but it maps "+
					"to no single argument and so cannot be shown as a diff",
				r.Name, strings.Join(unresolved, ", "))
		}
	}

	if isImport && len(r.RedactedKeys) > 0 {
		p.GetLogger(ctx).Warningf(
			"resource %q was imported without its credentials: the server withholds %s. "+
				"Set the matching arguments in your program, or the next update will clear them.",
			r.Name, strings.Join(r.RedactedKeys, ", "))
	}

	return infer.ReadResponse[ResourceArgs, ResourceState]{
		ID:     req.ID,
		Inputs: inputs,
		State:  ResourceState{ResourceArgs: inputs, AppliedDigests: appliedDigests},
	}, nil
}

// driftedDigestKeys returns the keys whose fingerprint no longer matches the one
// recorded at the last write — either because the value behind it changed, or
// because the server now holds a secret where it recorded none.
//
// A key that only appeared since the last write used to be dismissed as "not
// drift". It is drift: someone setting a password in the UI where there was
// none is exactly the change an operator needs to see. The wholesale
// empty-map guard at the call site is what keeps an import or a first refresh
// after upgrade from reporting every key as new.
//
// Keys that disappeared are not reported: the secret is gone, there is nothing
// to reconcile, and the ordinary config diff already covers a program removing
// one.
func driftedDigestKeys(applied, current map[string]string) []string {
	var drifted []string
	for k, cur := range current {
		if prev, ok := applied[k]; !ok || prev != cur {
			drifted = append(drifted, k)
		}
	}
	sort.Strings(drifted)
	return drifted
}

func (*Resource) Delete(ctx context.Context, req infer.DeleteRequest[ResourceState]) (infer.DeleteResponse, error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	return infer.DeleteResponse{}, c.DeleteResource(ctx, req.ID, req.State.Name)
}
