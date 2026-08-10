package adaptive

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Read implementations.
//
// Without these, infer falls back to returning whatever state it was handed, so
// `pulumi refresh` is a no-op and `pulumi import` cannot hydrate a resource from
// an existing id. Returning an empty ID tells the engine the resource is gone,
// which is what lets a delete performed in the Adaptive console show up as a
// recreate rather than as a failed update against a dead id.
//
// Credentials are never refreshed. The backend withholds them - resource
// configuration values are redacted, script bodies are omitted outright - so
// those properties keep whatever the program last declared. Refreshing them from
// an absent value would propose clearing a live credential on the next up.

// Compile-time proof that every resource actually satisfies CustomRead. A typo
// in a receiver or signature would otherwise fail silently: infer would fall
// back to the state-passthrough default and refresh would quietly do nothing.
var (
	_ infer.CustomRead[ResourceArgs, ResourceState]               = (*Resource)(nil)
	_ infer.CustomRead[EndpointArgs, EndpointState]               = (*Endpoint)(nil)
	_ infer.CustomRead[AuthorizationArgs, AuthorizationState]     = (*Authorization)(nil)
	_ infer.CustomRead[GroupArgs, GroupState]                     = (*Group)(nil)
	_ infer.CustomRead[ScriptArgs, ScriptState]                   = (*Script)(nil)
	_ infer.CustomRead[MSTeamsWorkflowArgs, MSTeamsWorkflowState] = (*MSTeamsWorkflow)(nil)
	_ infer.CustomRead[ScheduleArgs, ScheduleState]               = (*Schedule)(nil)
)

// ---------------------------------------------------------------------------
// Endpoint
// ---------------------------------------------------------------------------

func (*Endpoint) Read(ctx context.Context, req infer.ReadRequest[EndpointArgs, EndpointState]) (infer.ReadResponse[EndpointArgs, EndpointState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[EndpointArgs, EndpointState]{}, err
	}

	endpoint, err := c.GetSession(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[EndpointArgs, EndpointState]{}, err
	}
	if endpoint == nil {
		// Empty ID marks the resource as gone.
		return infer.ReadResponse[EndpointArgs, EndpointState]{}, nil
	}

	args := req.Inputs
	args.Name = endpoint.Name
	args.Resource = endpoint.Resource
	args.Authorization = optString(endpoint.Authorization)
	args.Cluster = optString(endpoint.Cluster)
	args.TTL = optString(endpoint.TTL)
	args.IdleTimeout = optString(endpoint.IdleTimeout)
	args.PauseTimeout = optString(endpoint.PauseTimeout)
	args.Memory = optString(endpoint.Memory)
	args.CPU = optString(endpoint.CPU)
	args.IsJitEnabled = &endpoint.IsJITEnabled
	args.ScriptOnlyAccess = &endpoint.ScriptOnlyAccess
	args.Users = endpoint.SessionUsers
	args.Groups = endpoint.Groups
	args.JitApprovers = endpoint.AccessApprovers
	args.Tags = endpoint.UserTags

	// Type is deliberately not refreshed. The input takes the user-facing
	// spelling ("direct", "cli", "client", "services") but the backend stores the
	// mapped value - "direct" is written as "cli" - so writing sessionType back
	// would show drift on every endpoint declared as "direct".

	return infer.ReadResponse[EndpointArgs, EndpointState]{
		ID:     req.ID,
		Inputs: args,
		State:  EndpointState{EndpointArgs: args},
	}, nil
}

// ---------------------------------------------------------------------------
// Authorization
// ---------------------------------------------------------------------------

func (*Authorization) Read(ctx context.Context, req infer.ReadRequest[AuthorizationArgs, AuthorizationState]) (infer.ReadResponse[AuthorizationArgs, AuthorizationState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, err
	}

	auth, err := c.GetAuthorization(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, err
	}
	if auth == nil {
		return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{}, nil
	}

	args := req.Inputs
	args.Name = auth.Name
	args.ResourceType = auth.ResourceType
	args.Description = optString(auth.Description)
	if auth.Permissions != "" {
		args.Permissions = auth.Permissions
	}

	return infer.ReadResponse[AuthorizationArgs, AuthorizationState]{
		ID:     req.ID,
		Inputs: args,
		State:  AuthorizationState{AuthorizationArgs: args},
	}, nil
}

// ---------------------------------------------------------------------------
// Group
// ---------------------------------------------------------------------------

func (*Group) Read(ctx context.Context, req infer.ReadRequest[GroupArgs, GroupState]) (infer.ReadResponse[GroupArgs, GroupState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[GroupArgs, GroupState]{}, err
	}

	team, err := c.GetTeam(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[GroupArgs, GroupState]{}, err
	}
	if team == nil {
		return infer.ReadResponse[GroupArgs, GroupState]{}, nil
	}

	args := req.Inputs
	args.Name = team.Name
	// Members come back as emails and endpoints as names, which is what the
	// write path accepts, so these compare directly against the program.
	args.Members = team.Members
	args.Endpoints = team.Endpoints

	return infer.ReadResponse[GroupArgs, GroupState]{
		ID:     req.ID,
		Inputs: args,
		State:  GroupState{GroupArgs: args},
	}, nil
}

// ---------------------------------------------------------------------------
// Script
// ---------------------------------------------------------------------------

func (*Script) Read(ctx context.Context, req infer.ReadRequest[ScriptArgs, ScriptState]) (infer.ReadResponse[ScriptArgs, ScriptState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[ScriptArgs, ScriptState]{}, err
	}

	script, err := c.GetScript(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[ScriptArgs, ScriptState]{}, err
	}
	if script == nil {
		return infer.ReadResponse[ScriptArgs, ScriptState]{}, nil
	}

	args := req.Inputs
	args.Name = script.Name
	args.Endpoint = script.Endpoint
	// Command is not refreshed: a script body can embed credentials, so the
	// backend withholds it. It keeps whatever the program last declared.

	return infer.ReadResponse[ScriptArgs, ScriptState]{
		ID:     req.ID,
		Inputs: args,
		State:  ScriptState{ScriptArgs: args},
	}, nil
}

// ---------------------------------------------------------------------------
// MSTeamsWorkflow
// ---------------------------------------------------------------------------

func (*MSTeamsWorkflow) Read(ctx context.Context, req infer.ReadRequest[MSTeamsWorkflowArgs, MSTeamsWorkflowState]) (infer.ReadResponse[MSTeamsWorkflowArgs, MSTeamsWorkflowState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[MSTeamsWorkflowArgs, MSTeamsWorkflowState]{}, err
	}

	resource, err := c.GetResource(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[MSTeamsWorkflowArgs, MSTeamsWorkflowState]{}, err
	}
	if resource == nil {
		return infer.ReadResponse[MSTeamsWorkflowArgs, MSTeamsWorkflowState]{}, nil
	}

	args := req.Inputs
	args.Name = resource.Name
	// WebhookUrl is not refreshed: a Teams webhook URL carries its own
	// authentication token, so the backend redacts it like any other credential.

	return infer.ReadResponse[MSTeamsWorkflowArgs, MSTeamsWorkflowState]{
		ID:     req.ID,
		Inputs: args,
		State:  MSTeamsWorkflowState{MSTeamsWorkflowArgs: args},
	}, nil
}

// ---------------------------------------------------------------------------
// Resource
// ---------------------------------------------------------------------------

// resourceConfigKeyToProp is the inverse of buildIntegrationConfig: a stored
// config key mapped to the Pulumi property that wrote it.
//
// This stays tractable because the backend strips every credential before
// returning the blob, so only non-secret keys need an inverse, and almost all of
// them map 1:1.
var resourceConfigKeyToProp = map[string]string{
	"hostname":     "host",
	"host":         "host",
	"port":         "port",
	"username":     "username",
	"databaseName": "databaseName",
	"sslMode":      "sslMode",
	"uri":          "uri",
	"protocol":     "protocol",

	"apiserver": "apiServer",
	"namespace": "namespace",

	"region":          "region",
	"project_id":      "projectId",
	"domain":          "domain",
	"clientID":        "clientId",
	"applicationID":   "applicationId",
	"tenantID":        "tenantId",
	"apiClientID":     "apiClientId",
	"organization_id": "organizationId",
	"public_key":      "publicKey",

	"warehouse":  "warehouse",
	"schema":     "schema",
	"role":       "role",
	"clientcert": "clientcert",

	"urls":  "urls",
	"hosts": "hosts",
	"user":  "defaultUser",
	"index": "index",

	"dd_site":              "ddSite",
	"image":                "image",
	"service_account_name": "serviceAccountName",
	"network_id":           "networkId",
	"webui_port":           "webuiPort",
	"use_proxy":            "useProxy",
	"use_service_account":  "useServiceAccount",
	"create_if_not_exists": "createIfNotExists",
	"useTenant":            "useTenant",
	"applicationName":      "applicationName",
	"subSystemName":        "subSystemName",
	"tokenID":              "tokenId",

	"arn":       "arn",
	"secret_id": "secretId",
	"aws_arn":   "awsArn",

	"useRdsIam":         "useRdsIam",
	"useIrsa":           "useIrsa",
	"awsRegion":         "awsRegion",
	"awsRoleArn":        "awsRoleArn",
	"awsAccessKeyId":    "awsAccessKeyId",
	"awsServiceAccount": "awsServiceAccount",
}

// resourceConfigKeyPropOverrides lists the keys whose meaning depends on the
// integration type. Getting one of these wrong would not fail loudly; it would
// write a value into the wrong property and show drift forever.
var resourceConfigKeyPropOverrides = map[string]map[string]string{
	"snowflake": {
		"databaseAccount":  "hostname",
		"databaseUsername": "username",
	},

	"aruba_sw":         {"hostname": "hostname"},
	"azuresqlserver":   {"hostname": "hostname"},
	"cisco_ngfw":       {"hostname": "hostname", "login_url": "uri"},
	"fortinet_ngfw":    {"hostname": "hostname", "login_url": "uri"},
	"hpe_switch":       {"hostname": "hostname", "login_url": "uri"},
	"paloalto_ngfw":    {"hostname": "hostname", "login_url": "loginUrl"},
	"syslog":           {"hostname": "hostname"},
	"rdp_windows":      {"hostname": "hostname"},
	"aruba_instant_on": {"host": "host"},

	"coralogix":           {"url": "uri"},
	"elasticsearch":       {"url": "uri"},
	"custom_siem_webhook": {"url": "uri"},
	"rabbitmq":            {"url": "uri"},
	"splunk":              {"url": "url"},

	"azurecosmosnosql": {"endpoint": "uri"},

	"aws":               {"aws_region_name": "regionName"},
	"awssecretsmanager": {"aws_region_name": "awsRegionName"},
}

// resourceConfigProp resolves one stored config key to a Pulumi property for a
// given integration type.
func resourceConfigProp(integrationType, key string) (string, bool) {
	if overrides, ok := resourceConfigKeyPropOverrides[integrationType]; ok {
		if prop, ok := overrides[key]; ok {
			return prop, true
		}
	}
	prop, ok := resourceConfigKeyToProp[key]
	return prop, ok
}

func (*Resource) Read(ctx context.Context, req infer.ReadRequest[ResourceArgs, ResourceState]) (infer.ReadResponse[ResourceArgs, ResourceState], error) {
	c, err := clientFromConfig(ctx)
	if err != nil {
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, err
	}

	resource, err := c.GetResource(ctx, req.ID)
	if err != nil {
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, err
	}
	if resource == nil {
		return infer.ReadResponse[ResourceArgs, ResourceState]{}, nil
	}

	args := req.Inputs
	args.Name = resource.Name
	// The backend rewrites "services" to "servicelist" on the way in; map it back
	// so a program declaring type "services" does not show drift.
	if resource.IntegrationType == "servicelist" {
		args.Type = "services"
	} else {
		args.Type = resource.IntegrationType
	}
	args.Tags = resource.UserTags
	if resource.DefaultCluster != "" {
		args.DefaultCluster = optString(resource.DefaultCluster)
	}

	// Properties whose values were withheld must keep their prior value.
	withheld := make(map[string]bool, len(resource.RedactedKeys))
	for _, key := range resource.RedactedKeys {
		if prop, ok := resourceConfigProp(resource.IntegrationType, key); ok {
			withheld[prop] = true
		}
	}

	for key, value := range resource.Configuration {
		prop, ok := resourceConfigProp(resource.IntegrationType, key)
		if !ok || withheld[prop] {
			// Unknown keys are expected: the stored blob carries fields the
			// provider never wrote (`version`, `name`, `type`).
			continue
		}
		if err := setPulumiProp(&args, prop, value); err != nil {
			return infer.ReadResponse[ResourceArgs, ResourceState]{},
				fmt.Errorf("refreshing %s: %w", prop, err)
		}
	}

	return infer.ReadResponse[ResourceArgs, ResourceState]{
		ID:     req.ID,
		Inputs: args,
		State:  ResourceState{ResourceArgs: args},
	}, nil
}

// setPulumiProp assigns value to the field of args carrying the given pulumi
// tag name. Reflection keeps this in step with ResourceArgs: adding a property
// to the struct makes it refreshable as soon as it appears in the key table,
// with no second list to update.
func setPulumiProp(args *ResourceArgs, prop string, value interface{}) error {
	v := reflect.ValueOf(args).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("pulumi")
		if tag == "" {
			continue
		}
		if name, _, _ := strings.Cut(tag, ","); name != prop {
			continue
		}
		return assignValue(v.Field(i), value)
	}
	// A key mapped to a property this struct does not have is a table bug, but
	// not a reason to fail an entire refresh.
	return nil
}

// assignValue writes a decoded JSON value into a *string, *bool, string or
// []string field, converting the shapes JSON decoding produces.
func assignValue(field reflect.Value, value interface{}) error {
	switch field.Interface().(type) {
	case *string:
		s := scalarToString(value)
		field.Set(reflect.ValueOf(&s))
	case string:
		field.SetString(scalarToString(value))
	case *bool:
		b, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected a bool, got %T", value)
		}
		field.Set(reflect.ValueOf(&b))
	case bool:
		b, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected a bool, got %T", value)
		}
		field.SetBool(b)
	case []string:
		field.Set(reflect.ValueOf(toStringSlice(value)))
	default:
		return fmt.Errorf("unsupported field type %s", field.Type())
	}
	return nil
}

// scalarToString normalises a decoded scalar. Ports round-trip as strings in the
// schema but decode as numbers, so a plain type assertion is not enough.
func scalarToString(value interface{}) string {
	switch t := value.(type) {
	case string:
		return t
	case float64:
		// JSON numbers decode as float64; render integral values without a
		// trailing ".0" so a port reads as "5432", not "5432.0".
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// toStringSlice handles both a real sequence and the newline-joined form
// serverlist uses for `hosts`.
func toStringSlice(value interface{}) []string {
	switch t := value.(type) {
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, v := range t {
			out = append(out, scalarToString(v))
		}
		return out
	case string:
		var out []string
		for _, line := range strings.Split(t, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				out = append(out, line)
			}
		}
		return out
	default:
		return nil
	}
}

// optString returns nil for an empty string so an absent value stays absent
// rather than becoming an explicit "".
func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
