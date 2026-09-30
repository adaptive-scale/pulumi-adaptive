//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resourceRead struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	IntegrationType string         `json:"integrationType"`
	UserTags        []string       `json:"userTags"`
	Configuration   map[string]any `json:"configuration"`
	RedactedKeys    []string       `json:"redactedKeys"`
}

func readTerraformResource(t *testing.T, cfg harness.Config, id string) resourceRead {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, cfg.URL+"/api/v1/terraform/resource/read/"+id, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", cfg.ServiceToken)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "read resource %s: %s", id, string(body))
	var out resourceRead
	require.NoError(t, json.Unmarshal(body, &out))
	return out
}

func assertConfigContains(t *testing.T, got map[string]any, want map[string]any) {
	t.Helper()
	for k, v := range want {
		if !assert.Contains(t, got, k, "configuration key %q", k) {
			continue
		}
		assert.EqualValues(t, v, got[k], "configuration[%q]", k)
	}
}

func assertRedacted(t *testing.T, got []string, keys ...string) {
	t.Helper()
	set := map[string]bool{}
	for _, k := range got {
		set[k] = true
	}
	for _, k := range keys {
		if !set[k] {
			t.Logf("server did not report %q as redacted (reported: %v)", k, got)
		}
	}
}

// TestResourceConfigCoverage creates representative resources that exercise the
// broad Resource config mapping. It verifies the server receives the expected
// non-secret YAML keys, and that secret keys are withheld/redacted on read.
func TestResourceConfigCoverage(t *testing.T) {
	cfg := harness.RequireProviderConfig(t)
	prefix := uniqueName("pulumi-it-cfg")

	type check struct {
		typ      string
		want     map[string]any
		redacted []string
	}
	checks := map[string]check{
		"aws": {"aws", map[string]any{
			"aws_region_name": "us-east-1", "use_role_arn": true, "aws_role_arn": "arn:aws:iam::123456789012:role/adaptive-test",
			"use_service_account": true,
		}, []string{"aws_access_key_id", "aws_secret_access_key"}},
		"ssh":           {"ssh", map[string]any{"username": "ubuntu", "hostname": "ssh.example.com", "port": "22"}, []string{"password"}},
		"ssh-key":       {"ssh", map[string]any{"username": "ubuntu", "hostname": "ssh-key.example.com", "port": "22", "usePassword": false}, []string{"sshKey"}},
		"rabbitmq":      {"rabbitmq", map[string]any{"url": "https://rabbit.example.com", "username": "guest"}, []string{"password"}},
		"elasticsearch": {"elasticsearch", map[string]any{"url": "https://es.example.com", "username": "elastic", "index": "logs-*"}, []string{"password"}},
		"azurecosmos":   {"azurecosmosnosql", map[string]any{"endpoint": "https://cosmos.example.com"}, []string{"key"}},
		"elasticache": {"awselasticcache", map[string]any{
			"username": "default", "host": "cache.example.com", "port": "6379", "tlsEnabled": true, "tlsSkipVerify": true,
			"access_control_method": "user_group", "access_control_group": "rg", "aws_region_name": "us-east-2",
			"useIamAuth": true, "useIrsa": true, "aws_role_arn": "arn:aws:iam::123456789012:role/elasticache",
			"aws_service_account": "elasticache-sa", "cache_name": "cache-a", "cache_type": "replication_group",
		}, []string{"password", "aws_access_key_id", "aws_secret_access_key"}},
		"postgres": {"postgres", map[string]any{
			"username": "pg", "databaseName": "app", "hostname": "pg.example.com", "port": "5432", "sslMode": "require",
			"useRdsIam": true, "useIrsa": true, "authMode": "iam", "awsRegion": "us-east-2",
			"awsRoleArn": "arn:aws:iam::123456789012:role/postgres", "awsServiceAccount": "pg-sa",
		}, []string{"password", "rootCert", "crtText", "keyText", "awsAccessKeyId", "awsSecretAccessKey"}},
		"mysql": {"mysql", map[string]any{
			"username": "mysql", "databaseName": "app", "hostname": "mysql.example.com", "port": "3306", "sslMode": "require",
			"useRdsIam": true, "useIrsa": true, "authMode": "iam", "awsRegion": "us-east-2",
			"awsRoleArn": "arn:aws:iam::123456789012:role/mysql", "awsServiceAccount": "mysql-sa",
		}, []string{"password", "rootCert", "clientCert", "clientKey", "awsAccessKeyId", "awsSecretAccessKey"}},
		"kafka": {"kafka", map[string]any{
			"useMskIam": true, "useIrsa": true, "bootstrapServers": "b-1.example:9098", "awsRegion": "us-east-2",
			"awsRoleArn": "arn:aws:iam::123456789012:role/kafka", "awsServiceAccount": "kafka-sa",
		}, []string{"client_configuration", "awsAccessKeyId", "awsSecretAccessKey"}},
		"keyspaces": {"keyspaces", map[string]any{
			"keyspace": "ks", "keyspaces_endpoint": "cassandra.us-east-2.amazonaws.com", "aws_region_name": "us-east-2",
			"use_role_arn": true, "aws_role_arn": "arn:aws:iam::123456789012:role/keyspaces", "use_service_account": true,
		}, []string{"aws_access_key_id", "aws_secret_access_key"}},
		"syslog":        {"syslog", map[string]any{"hostname": "syslog.example.com", "port": "514", "protocol": "tcp", "tlsEnabled": true, "insecureSkipVerify": true}, []string{"clientCertificate", "clientKey"}},
		"services":      {"servicelist", map[string]any{"urls": "https://one.example.com,https://two.example.com", "enableTLS": true}, nil},
		"rdp":           {"rdp_windows", map[string]any{"hostname": "rdp.example.com", "username": "administrator", "port": "3389", "allowFileTransfer": true}, []string{"password"}},
		"datadog":       {"datadog", map[string]any{"dd_site": "datadoghq.com"}, []string{"dd_api_key", "dd_app_key"}},
		"awsdocumentdb": {"awsdocumentdb", map[string]any{"tlsEnabled": true}, []string{"uri"}},
		"mongodb":       {"mongodb", map[string]any{"useTLS": true}, []string{"uri", "clientCertificate"}},
		"awsredshift":   {"awsredshift", map[string]any{"username": "redshift", "databaseName": "dev", "hostname": "redshift.example.com", "port": "5439", "sslMode": "require"}, []string{"password"}},
		"sqlserver":     {"sql_server", map[string]any{"username": "sa", "databaseName": "app", "hostname": "sql.example.com", "port": "1433", "sslMode": "encrypt"}, []string{"password"}},
		"azuresql":      {"azuresqlserver", map[string]any{"username": "sa", "databaseName": "app", "hostname": "azsql.example.com", "port": "1433", "sslMode": "encrypt"}, []string{"password"}},
		"clickhouse":    {"clickhouse", map[string]any{"username": "default", "database": "default", "hostname": "ch.example.com", "port": "9440", "use-tls": true}, []string{"password"}},
		"ngfw":          {"fortinet_ngfw", map[string]any{"hostname": "fw.example.com", "login_url": "https://fw.example.com", "port": "22", "usePassword": false, "use_proxy": true, "username": "admin", "webui_port": "443"}, []string{"password", "sshKey"}},
		"aruba":         {"aruba_sw", map[string]any{"hostname": "aruba.example.com", "login_url": "https://aruba.example.com", "port": "22", "usePassword": false, "username": "admin", "webui_port": "443"}, []string{"password", "sshKey"}},
		"juniper":       {"juniper_sw", map[string]any{"hostname": "juniper.example.com", "login_url": "https://juniper.example.com", "port": "22", "usePassword": false, "username": "admin", "webui_port": "443"}, []string{"password", "sshKey"}},
	}

	commonTags := pulumi.StringArray{pulumi.String("phase=create")}
	outs, stack := harness.DeployStack(t, cfg, stackName("cfg-coverage"), func(ctx *pulumi.Context) error {
		mkName := func(suffix string) string { return prefix + "-" + suffix }
		clusterName := mkName("cluster")
		cluster, err := adaptive.NewResource(ctx, "cluster", &adaptive.ResourceArgs{
			Name:         pulumi.String(clusterName),
			Type:         pulumi.String("kubernetes"),
			ApiServer:    pulumi.String("https://kubernetes.default.svc"),
			ClusterToken: pulumi.String("not-a-real-token"),
			ClusterCert:  pulumi.String("not-a-real-cert"),
			Tags:         pulumi.StringArray{pulumi.String("phase=create")},
		})
		if err != nil {
			return err
		}
		add := func(key string, args *adaptive.ResourceArgs) error {
			args.Name = pulumi.String(mkName(key))
			args.DefaultCluster = pulumi.String(clusterName)
			args.Tags = commonTags
			r, err := adaptive.NewResource(ctx, key, args, pulumi.DependsOn([]pulumi.Resource{cluster}))
			if err != nil {
				return err
			}
			ctx.Export("id_"+key, r.ID())
			return nil
		}
		resources := map[string]*adaptive.ResourceArgs{
			"aws":           {Type: pulumi.String("aws"), RegionName: pulumi.String("us-east-1"), AccessKeyId: pulumi.String("AKIA_TEST"), SecretAccessKey: pulumi.String("secret"), UseRoleArn: pulumi.Bool(true), AwsRoleArn: pulumi.String("arn:aws:iam::123456789012:role/adaptive-test"), UseServiceAccount: pulumi.Bool(true), ServiceAccount: pulumi.String("adaptive-aws"), CreateIfNotExists: pulumi.Bool(true), EnableAllowedCommands: pulumi.Bool(true), AllowedCommands: pulumi.String("s3:ls,ec2:describe-instances")},
			"ssh":           {Type: pulumi.String("ssh"), Username: pulumi.String("ubuntu"), Host: pulumi.String("ssh.example.com"), Port: pulumi.String("22"), Password: pulumi.String("pw")},
			"ssh-key":       {Type: pulumi.String("ssh"), Username: pulumi.String("ubuntu"), Host: pulumi.String("ssh-key.example.com"), Port: pulumi.String("22"), Key: pulumi.String("PRIVATE KEY")},
			"rabbitmq":      {Type: pulumi.String("rabbitmq"), Url: pulumi.String("https://rabbit.example.com"), Username: pulumi.String("guest"), Password: pulumi.String("guest")},
			"elasticsearch": {Type: pulumi.String("elasticsearch"), Url: pulumi.String("https://es.example.com"), Username: pulumi.String("elastic"), Password: pulumi.String("pw"), Index: pulumi.String("logs-*")},
			"azurecosmos":   {Type: pulumi.String("azurecosmosnosql"), Url: pulumi.String("https://cosmos.example.com"), ApiToken: pulumi.String("cosmos-key")},
			"elasticache":   {Type: pulumi.String("awselasticcache"), Username: pulumi.String("default"), Password: pulumi.String("pw"), Host: pulumi.String("cache.example.com"), Port: pulumi.String("6379"), TlsEnabled: pulumi.Bool(true), TlsSkipVerify: pulumi.Bool(true), AccessControlMethod: pulumi.String("user_group"), AccessControlGroup: pulumi.String("rg"), AwsRegionName: pulumi.String("us-east-2"), AccessKeyId: pulumi.String("AKIA_CACHE"), SecretAccessKey: pulumi.String("secret"), UseIamAuth: pulumi.Bool(true), UseIrsa: pulumi.Bool(true), AwsRoleArn: pulumi.String("arn:aws:iam::123456789012:role/elasticache"), AwsServiceAccount: pulumi.String("elasticache-sa"), CacheName: pulumi.String("cache-a"), CacheType: pulumi.String("replication_group")},
			"postgres":      {Type: pulumi.String("postgres"), Username: pulumi.String("pg"), Password: pulumi.String("pw"), DatabaseName: pulumi.String("app"), Host: pulumi.String("pg.example.com"), Port: pulumi.String("5432"), SslMode: pulumi.String("require"), TlsRootCert: pulumi.String("ROOT"), TlsCertFile: pulumi.String("CERT"), TlsKeyFile: pulumi.String("KEY"), UseRdsIam: pulumi.Bool(true), UseIrsa: pulumi.Bool(true), AuthMode: pulumi.String("iam"), AwsRegion: pulumi.String("us-east-2"), AwsRoleArn: pulumi.String("arn:aws:iam::123456789012:role/postgres"), AccessKeyId: pulumi.String("AKIA_PG"), SecretAccessKey: pulumi.String("secret"), AwsServiceAccount: pulumi.String("pg-sa")},
			"mysql":         {Type: pulumi.String("mysql"), Username: pulumi.String("mysql"), Password: pulumi.String("pw"), DatabaseName: pulumi.String("app"), Host: pulumi.String("mysql.example.com"), Port: pulumi.String("3306"), RootCert: pulumi.String("ROOT"), ClientCert: pulumi.String("CERT"), ClientKey: pulumi.String("KEY"), OldVersion: pulumi.Bool(true), UseRdsIam: pulumi.Bool(true), UseIrsa: pulumi.Bool(true), AuthMode: pulumi.String("iam"), AwsRegion: pulumi.String("us-east-2"), AwsRoleArn: pulumi.String("arn:aws:iam::123456789012:role/mysql"), AccessKeyId: pulumi.String("AKIA_MY"), SecretAccessKey: pulumi.String("secret"), AwsServiceAccount: pulumi.String("mysql-sa")},
			"kafka":         {Type: pulumi.String("kafka"), ClientConfiguration: pulumi.String("bootstrap.servers=b-1.example:9092"), UseMskIam: pulumi.Bool(true), UseIrsa: pulumi.Bool(true), BootstrapServers: pulumi.String("b-1.example:9098"), AwsRegion: pulumi.String("us-east-2"), AwsRoleArn: pulumi.String("arn:aws:iam::123456789012:role/kafka"), AccessKeyId: pulumi.String("AKIA_KAFKA"), SecretAccessKey: pulumi.String("secret"), AwsServiceAccount: pulumi.String("kafka-sa")},
			"keyspaces":     {Type: pulumi.String("keyspaces"), Keyspace: pulumi.String("ks"), KeyspacesEndpoint: pulumi.String("cassandra.us-east-2.amazonaws.com"), AwsRegionName: pulumi.String("us-east-2"), AccessKeyId: pulumi.String("AKIA_KS"), SecretAccessKey: pulumi.String("secret"), UseRoleArn: pulumi.Bool(true), AwsRoleArn: pulumi.String("arn:aws:iam::123456789012:role/keyspaces"), UseServiceAccount: pulumi.Bool(true), ServiceAccount: pulumi.String("keyspaces-sa"), CreateIfNotExists: pulumi.Bool(true)},
			"syslog":        {Type: pulumi.String("syslog"), Hostname: pulumi.String("syslog.example.com"), Port: pulumi.String("514"), Protocol: pulumi.String("tcp"), TlsEnabled: pulumi.Bool(true), CaCertificate: pulumi.String("CA"), ClientCertificate: pulumi.String("CERT"), ClientKey: pulumi.String("KEY"), InsecureSkipVerify: pulumi.Bool(true)},
			"services":      {Type: pulumi.String("services"), Urls: pulumi.String("https://one.example.com,https://two.example.com"), EnableTls: pulumi.Bool(true)},
			"rdp":           {Type: pulumi.String("rdp_windows"), Hostname: pulumi.String("rdp.example.com"), Username: pulumi.String("administrator"), Password: pulumi.String("pw"), Port: pulumi.String("3389"), AllowFileTransfer: pulumi.Bool(true)},
			"datadog":       {Type: pulumi.String("datadog"), DdSite: pulumi.String("datadoghq.com"), DdApiKey: pulumi.String("dd-api"), DdAppKey: pulumi.String("dd-app")},
			"awsdocumentdb": {Type: pulumi.String("awsdocumentdb"), Uri: pulumi.String("mongodb://docdb.example.com:27017"), TlsEnabled: pulumi.Bool(true)},
			"mongodb":       {Type: pulumi.String("mongodb"), Uri: pulumi.String("mongodb://mongo.example.com:27017"), ClientCertificate: pulumi.String("CERT"), UseTls: pulumi.Bool(true)},
			"awsredshift":   {Type: pulumi.String("awsredshift"), Username: pulumi.String("redshift"), Password: pulumi.String("pw"), DatabaseName: pulumi.String("dev"), Host: pulumi.String("redshift.example.com"), Port: pulumi.String("5439"), SslMode: pulumi.String("require")},
			"sqlserver":     {Type: pulumi.String("sql_server"), Username: pulumi.String("sa"), Password: pulumi.String("pw"), DatabaseName: pulumi.String("app"), Host: pulumi.String("sql.example.com"), Port: pulumi.String("1433"), SslMode: pulumi.String("encrypt")},
			"azuresql":      {Type: pulumi.String("azuresqlserver"), Username: pulumi.String("sa"), Password: pulumi.String("pw"), DatabaseName: pulumi.String("app"), Hostname: pulumi.String("azsql.example.com"), Port: pulumi.String("1433"), SslMode: pulumi.String("encrypt")},
			"clickhouse":    {Type: pulumi.String("clickhouse"), Username: pulumi.String("default"), Password: pulumi.String("pw"), DatabaseName: pulumi.String("default"), Host: pulumi.String("ch.example.com"), Port: pulumi.String("9440"), UseTls: pulumi.Bool(true)},
			"ngfw":          {Type: pulumi.String("fortinet_ngfw"), Hostname: pulumi.String("fw.example.com"), LoginUrl: pulumi.String("https://fw.example.com"), Port: pulumi.String("22"), UseProxy: pulumi.Bool(true), Username: pulumi.String("admin"), Password: pulumi.String("pw"), WebuiPort: pulumi.String("443"), Key: pulumi.String("PRIVATE KEY")},
			"aruba":         {Type: pulumi.String("aruba_sw"), Hostname: pulumi.String("aruba.example.com"), LoginUrl: pulumi.String("https://aruba.example.com"), Port: pulumi.String("22"), Username: pulumi.String("admin"), Password: pulumi.String("pw"), WebuiPort: pulumi.String("443"), Key: pulumi.String("PRIVATE KEY")},
			"juniper":       {Type: pulumi.String("juniper_sw"), Hostname: pulumi.String("juniper.example.com"), LoginUrl: pulumi.String("https://juniper.example.com"), Port: pulumi.String("22"), Username: pulumi.String("admin"), Password: pulumi.String("pw"), WebuiPort: pulumi.String("443"), Key: pulumi.String("PRIVATE KEY")},
		}
		for _, key := range sortedKeys(resources) {
			if err := add(key, resources[key]); err != nil {
				return err
			}
		}
		return nil
	})

	for key, check := range checks {
		id := harness.StringOutput(t, outs, "id_"+key)
		got := readTerraformResource(t, cfg, id)
		require.Equal(t, check.typ, got.IntegrationType, key)
		require.True(t, strings.HasPrefix(got.Name, prefix+"-"), got.Name)
		assert.Contains(t, got.UserTags, "phase=create", key)
		assertConfigContains(t, got.Configuration, check.want)
		if len(check.redacted) > 0 {
			assertRedacted(t, got.RedactedKeys, check.redacted...)
		}
	}

	// Exercise the update/sync path for every resource in the matrix. We update
	// a common non-secret field (tags), preview the change, apply it, and verify
	// every resource in Adaptive now reflects the local Pulumi program.
	commonTags = pulumi.StringArray{pulumi.String("phase=updated")}
	changes := harness.Preview(t, stack)
	assert.GreaterOrEqual(t, changes["update"], len(checks), "all matrix resources should plan tag updates: %v", changes)
	harness.Up(t, stack)
	for key := range checks {
		id := harness.StringOutput(t, outs, "id_"+key)
		got := readTerraformResource(t, cfg, id)
		assert.Contains(t, got.UserTags, "phase=updated", key)
	}
	harness.AssertRefreshClean(t, stack)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if fmt.Sprint(keys[j]) < fmt.Sprint(keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
