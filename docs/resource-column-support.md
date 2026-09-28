## `1password`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `service_account` | `serviceAccount` | No | `serviceAccount` |  |
| `use_connect_server` | `useConnectServer` | No | `useConnectServer` |  |
| `connect_server_url` | `connectServerUrl` | No | `connectServerUrl` |  |

## `adaptive_rdp`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `targets` | `targets` | Yes | `targets` |  |

## `adaptiveencrypt`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `value` | `value` | No | `value` |  |

## `adaptiveremotedesktop`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `cpu` | `cpu` | No | `cpu` |  |
| `memory` | `memory` | No | `memory` |  |
| `storage` | `storage` | No | `storage` |  |

## `aruba_instant_on`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `sshKey` | `key` | Yes | `key` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |

## `aruba_sw`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `sshKey` | `key` | Yes | `key` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |

## `avastbusinessedr`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `clientID` | `clientId` | No | `clientId` |  |
| `clientSecret` | `clientSecret` | Yes | `clientSecret` |  |

## `aws`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `version` | — | — | — | "1.0" |
| `aws_region_name` | `regionName` | No | `regionName` |  |
| `aws_access_key_id` | `accessKeyId` | Yes | `accessKeyId` |  |
| `aws_secret_access_key` | `secretAccessKey` | Yes | `secretAccessKey` |  |
| `use_role_arn` | `useRoleArn` | No | `useRoleArn` |  |
| `aws_role_arn` | `awsRoleArn` | No | `awsRoleArn` |  |
| `use_service_account` | `useServiceAccount` | No | `useServiceAccount` |  |
| `service_account` | `serviceAccount` | No | `serviceAccount` |  |
| `create_if_not_exists` | `createIfNotExists` | No | `createIfNotExists` |  |
| `enable_allowed_commands` | `enableAllowedCommands` | No | `enableAllowedCommands` |  |
| `allowed_commands` | `allowedCommands` | No | `allowedCommands` |  |

## `awscloudwatchlogs`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `aws_region` | `awsRegionName` | No | `awsRegionName` |  |
| `arn` | `arn` | No | `arn` |  |
| `log_group_name` | `logGroupName` | No | `logGroupName` |  |
| `log_stream_name` | `logStreamName` | No | `logStreamName` |  |

## `awsdocumentdb`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `uri` | `uri` | Yes | `uri` |  |
| `tlsEnabled` | `tlsEnabled` | No | `tlsEnabled` |  |

## `awsdocumentdb_aws_secret_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `arn` | `arn` | No | `arn` |  |
| `region` | `region` | No | `region` |  |
| `secret_id` | `secretId` | Yes | `secretId` |  |
| `tlsEnabled` | `tlsEnabled` | No | `tlsEnabled` |  |

## `awselasticcache`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `host` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `tlsEnabled` | `tlsEnabled` | No | `tlsEnabled` |  |
| `tlsSkipVerify` | `tlsSkipVerify` | No | `tlsSkipVerify` |  |
| `access_control_method` | `accessControlMethod` | No | `accessControlMethod` |  |
| `access_control_group` | `accessControlGroup` | No | `accessControlGroup` |  |
| `aws_region_name` | `awsRegionName` | No | `awsRegionName` |  |
| `aws_access_key_id` | `accessKeyId` | Yes | `accessKeyId` |  |
| `aws_secret_access_key` | `secretAccessKey` | Yes | `secretAccessKey` |  |
| `useIamAuth` | `useIamAuth` | No | `useIamAuth` |  |
| `useIrsa` | `useIrsa` | No | `useIrsa` |  |
| `aws_role_arn` | `awsRoleArn` | No | `awsRoleArn` |  |
| `aws_service_account` | `awsServiceAccount` | No | `awsServiceAccount` |  |
| `cache_name` | `cacheName` | No | `cacheName` |  |
| `cache_type` | `cacheType` | No | `cacheType` |  |

## `awsredshift`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |

## `awssecretsmanager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `aws_region_name` | `awsRegionName` | No | `awsRegionName` |  |
| `aws_arn` | `awsArn` | No | `awsArn` |  |

## `azure`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `tenantID` | `tenantId` | No | `tenantId` |  |
| `applicationID` | `applicationId` | No | `applicationId` |  |
| `clientSecret` | `clientSecret` | Yes | `clientSecret` |  |

## `azure_documentdb`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `uri` | `uri` | Yes | `uri` |  |

## `azureactivedirectory`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `domain` | `domain` | No | `domain` |  |
| `clientID` | `clientId` | No | `clientId` |  |
| `clientSecret` | `clientSecret` | Yes | `clientSecret` |  |
| `tenantID` | `tenantId` | No | `tenantId` |  |
| `useTenant` | `useTenant` | No | `useTenant` |  |

## `azurecosmosnosql`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `endpoint` | `url` | No | `url` |  |
| `key` | `apiToken` | Yes | `apiToken` |  |

## `azuresqlserver`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |

## `big_query`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `credential_json` | `credentialJson` | Yes | `credentialJson` |  |

## `chrome`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `url` | `url` | No | `url` |  |
| `prestart` | `prestart` | No | `prestart` |  |
| `automationMode` | `automationMode` | No | `automationMode` |  |
| `fields` | `fields` | No | `fields` |  |
| `script` | `script` | No | `script` |  |

## `cisco_ngfw`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `port` | `port` | No | `port` |  |
| `usePassword` | `key` | Yes | — |  |
| `use_proxy` | `useProxy` | No | `useProxy` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `version` | — | — | — | "1.0" |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `clickhouse`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `database` | `databaseName` | No | `databaseName` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `use-tls` | `useTls` | No | `useTls` |  |

## `cockroachdb`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |
| `rootCert` | `tlsRootCert` | Yes | `tlsRootCert` |  |

## `cockroachdb_aws_secrets_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `arn` | `arn` | No | `arn` |  |
| `region` | `region` | No | `region` |  |
| `secret_id` | `secretId` | Yes | `secretId` |  |
| `db_user` | `username` | No | `username` |  |
| `db_password` | `password` | Yes | `password` |  |
| `db_name` | `databaseName` | No | `databaseName` |  |
| `db_host` | `host` | No | `host` |  |
| `db_port` | `port` | No | `port` |  |
| `db_rootCert` | `rootCert` | Yes | `rootCert` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |

## `confluent`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `organization_id` | `organizationId` | No | `organizationId` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `resource` | `resource` | No | `resource` |  |
| `apiKey` | `apiKey` | Yes | `apiKey` |  |
| `apiSecret` | `apiSecret` | Yes | `apiSecret` |  |

## `coralogix`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `url` | `uri` | Yes | `uri` |  |
| `privateKey` | `privateKey` | Yes | `privateKey` |  |
| `applicationName` | `applicationName` | No | `applicationName` |  |
| `subSystemName` | `subSystemName` | No | `subSystemName` |  |

## `custom_siem_webhook`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `url` | `uri` | Yes | `uri` |  |
| `sharedSecret` | `sharedSecret` | Yes | `sharedSecret` |  |

## `customintegration`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `image` | `image` | No | `image` |  |
| `service_account_name` | `serviceAccountName` | No | `serviceAccountName` |  |

## `datadog`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `dd_site` | `ddSite` | No | `ddSite` |  |
| `dd_api_key` | `ddApiKey` | Yes | `ddApiKey` |  |
| `dd_app_key` | `ddAppKey` | Yes | `ddAppKey` |  |

## `digitalocean`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `api_token` | `apiToken` | Yes | `apiToken` |  |

## `elasticsearch`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `url` | `url` | No | `url` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `index` | `index` | No | `index` |  |

## `fortinet_analyzer`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `fortinet_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `fortinet_ngfw`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `port` | `port` | No | `port` |  |
| `type` | — | — | — | "fortinet_ngfw" |
| `usePassword` | `key` | Yes | — |  |
| `use_proxy` | `useProxy` | No | `useProxy` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `version` | — | — | — | "1.0" |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `gcp`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1" |
| `name` | `name` | No | — |  |
| `project_id` | `projectId` | No | `projectId` |  |
| `key_file` | `keyFile` | Yes | `keyFile` |  |

## `google`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `Version` | — | — | — | "1" |
| `name` | `name` | No | — |  |
| `domain` | `domain` | No | `domain` |  |
| `clientID` | `clientId` | No | `clientId` |  |
| `clientSecret` | `clientSecret` | Yes | `clientSecret` |  |

## `heroku`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `machine` | `machine` | No | `machine` |  |
| `username` | `username` | No | `username` |  |
| `token` | `token` | Yes | `token` |  |

## `hpe_switch`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `port` | `port` | No | `port` |  |
| `usePassword` | `key` | Yes | — |  |
| `use_proxy` | `useProxy` | No | `useProxy` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `version` | — | — | — | "1.0" |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `ivanti`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `use_proxy` | `useProxy` | No | `useProxy` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `jumpcloud`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `clientID` | `clientId` | No | `clientId` |  |
| `clientSecret` | `clientSecret` | Yes | `clientSecret` |  |
| `domain` | `domain` | No | `domain` |  |
| `apiKey` | `apiToken` | Yes | `apiToken` |  |

## `juniper_sw`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `kafka`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `client_configuration` | `clientConfiguration` | Yes | `clientConfiguration` |  |
| `useMskIam` | `useMskIam` | No | `useMskIam` |  |
| `useIrsa` | `useIrsa` | No | `useIrsa` |  |
| `bootstrapServers` | `bootstrapServers` | No | `bootstrapServers` |  |
| `awsRegion` | `awsRegion` | No | `awsRegion` |  |
| `awsRoleArn` | `awsRoleArn` | No | `awsRoleArn` |  |
| `awsAccessKeyId` | `accessKeyId` | Yes | `accessKeyId` |  |
| `awsSecretAccessKey` | `secretAccessKey` | Yes | `secretAccessKey` |  |
| `awsServiceAccount` | `awsServiceAccount` | No | `awsServiceAccount` |  |

## `keyspaces`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `use_service_account` | `useServiceAccount` | No | `useServiceAccount` |  |
| `create_if_not_exists` | `createIfNotExists` | No | `createIfNotExists` |  |
| `name` | `name` | No | — |  |
| `keyspace` | `keyspace` | No | `keyspace` |  |
| `keyspaces_endpoint` | `keyspacesEndpoint` | No | `keyspacesEndpoint` |  |
| `version` | — | — | — | "1.0" |
| `aws_region_name` | `awsRegionName` | No | `awsRegionName` |  |
| `aws_access_key_id` | `accessKeyId` | Yes | `accessKeyId` |  |
| `aws_secret_access_key` | `secretAccessKey` | Yes | `secretAccessKey` |  |
| `use_role_arn` | `useRoleArn` | No | `useRoleArn` |  |
| `aws_role_arn` | `awsRoleArn` | No | `awsRoleArn` |  |
| `service_account` | `serviceAccount` | No | `serviceAccount` |  |

## `kubernetes`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `apiserver` | `apiServer` | No | `apiServer` |  |
| `token` | `clusterToken` | Yes | `clusterToken` |  |
| `cacrt` | `clusterCert` | Yes | `clusterCert` |  |
| `namespace` | `namespace` | No | `namespace` |  |
| `tolerationsBytes` | `tolerations` | No | `tolerations` |  |
| `annotationsBytes` | `annotations` | No | `annotations` |  |
| `nodeSelectorBytes` | `nodeSelector` | No | `nodeSelector` |  |
| `affinityBytes` | `nodeAffinity` | No | `nodeAffinity` |  |

## `ldap`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `ldap-encryption-method` | `ldapEncryptionMethod` | No | `ldapEncryptionMethod` |  |
| `ldap-search-bind-dn` | `ldapSearchBindDn` | No | `ldapSearchBindDn` |  |
| `ldap-search-bind-password` | `ldapSearchBindPassword` | Yes | `ldapSearchBindPassword` |  |
| `ldap-user-name-attribute` | `ldapUserNameAttribute` | No | `ldapUserNameAttribute` |  |
| `ldap-user-base-dn` | `ldapUserBaseDn` | No | `ldapUserBaseDn` |  |

## `mongodb`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `uri` | `uri` | Yes | `uri` |  |
| `clientCertificate` | `clientCertificate` | Yes | `clientCertificate` |  |
| `useTLS` | `useTls` | No | `useTls` |  |

## `mongodb-do`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `dbname` | `databaseName` | No | `databaseName` |  |
| `api_token` | `apiToken` | Yes | `apiToken` |  |

## `mongodb36`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `uri` | `uri` | Yes | `uri` |  |
| `clientCertificate` | `clientCertificate` | Yes | `clientCertificate` |  |
| `useTLS` | `useTls` | No | `useTls` |  |

## `mongodb_atlas`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `uri` | `uri` | Yes | `uri` |  |
| `organization_id` | `organizationId` | No | `organizationId` |  |
| `public_key` | `publicKey` | Yes | `publicKey` |  |
| `private_key` | `privateKey` | Yes | `privateKey` |  |
| `project_id` | `projectId` | No | `projectId` |  |

## `mongodb_aws_secrets_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `arn` | `arn` | No | `arn` |  |
| `region` | `region` | No | `region` |  |
| `secret_id` | `secretId` | Yes | `secretId` |  |
| `key` | `key` | Yes | `key` |  |

## `msteams`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `appID` | `clientId` | No | `clientId` |  |
| `appKey` | `clientSecret` | Yes | `clientSecret` |  |
| `tenantID` | `tenantId` | No | `tenantId` |  |

## `msteams_workflow`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `webhookURL` | `webhookUrl` | Yes | `webhookUrl` |  |

## `mysql`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `sslMode` | — | — | — | "require" |
| `rootCert` | `rootCert` | Yes | `rootCert` |  |
| `clientCert` | `clientCert` | Yes | `clientCert` |  |
| `clientKey` | `clientKey` | Yes | `clientKey` |  |
| `oldVersion` | `oldVersion` | No | `oldVersion` |  |
| `useRdsIam` | `useRdsIam` | No | `useRdsIam` |  |
| `useIrsa` | `useIrsa` | No | `useIrsa` |  |
| `authMode` | `authMode` | No | `authMode` |  |
| `awsRegion` | `awsRegion` | No | `awsRegion` |  |
| `awsRoleArn` | `awsRoleArn` | No | `awsRoleArn` |  |
| `awsAccessKeyId` | `accessKeyId` | Yes | `accessKeyId` |  |
| `awsSecretAccessKey` | `secretAccessKey` | Yes | `secretAccessKey` |  |
| `awsServiceAccount` | `awsServiceAccount` | No | `awsServiceAccount` |  |

## `mysql_aws_secrets_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `arn` | `arn` | No | `arn` |  |
| `region` | `region` | No | `region` |  |
| `secret_id` | `secretId` | Yes | `secretId` |  |

## `okta`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `domain` | `domain` | No | `domain` |  |
| `clientID` | `clientId` | No | `clientId` |  |
| `clientSecret` | `clientSecret` | Yes | `clientSecret` |  |

## `onelogin`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `domain` | `domain` | No | `domain` |  |
| `clientID` | `clientId` | No | `clientId` |  |
| `clientSecret` | `clientSecret` | Yes | `clientSecret` |  |
| `apiClientID` | `apiClientId` | No | `apiClientId` |  |
| `apiClientSecret` | `apiClientSecret` | Yes | `apiClientSecret` |  |

## `oracle`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `service_name` | `serviceName` | No | `serviceName` |  |

## `paloalto_ngfw`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `password` | `password` | Yes | `password` |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `sshKey` | `key` | Yes | `key` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |

## `postgres`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |
| `rootCert` | `tlsRootCert` | Yes | `tlsRootCert` |  |
| `crtText` | `tlsCertFile` | Yes | `tlsCertFile` |  |
| `keyText` | `tlsKeyFile` | Yes | `tlsKeyFile` |  |
| `useRdsIam` | `useRdsIam` | No | `useRdsIam` |  |
| `useIrsa` | `useIrsa` | No | `useIrsa` |  |
| `authMode` | `authMode` | No | `authMode` |  |
| `awsRegion` | `awsRegion` | No | `awsRegion` |  |
| `awsRoleArn` | `awsRoleArn` | No | `awsRoleArn` |  |
| `awsAccessKeyId` | `accessKeyId` | Yes | `accessKeyId` |  |
| `awsSecretAccessKey` | `secretAccessKey` | Yes | `secretAccessKey` |  |
| `awsServiceAccount` | `awsServiceAccount` | No | `awsServiceAccount` |  |

## `postgres_aws_secrets_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `arn` | `arn` | No | `arn` |  |
| `region` | `region` | No | `region` |  |
| `secret_id` | `secretId` | Yes | `secretId` |  |

## `proxysql`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |
| `rootCert` | `rootCert` | Yes | `rootCert` |  |
| `clientCert` | `clientCert` | Yes | `clientCert` |  |
| `clientKey` | `clientKey` | Yes | `clientKey` |  |
| `oldVersion` | `oldVersion` | No | `oldVersion` |  |
| `proxysqlAdminPort` | `proxysqlAdminPort` | No | `proxysqlAdminPort` |  |
| `proxysqlAdminUsername` | `proxysqlAdminUsername` | No | `proxysqlAdminUsername` |  |
| `proxysqlAdminPassword` | `proxysqlAdminPassword` | Yes | `proxysqlAdminPassword` |  |
| `proxysqlHostgroupID` | `proxysqlHostgroupId` | No | `proxysqlHostgroupId` |  |

## `rabbitmq`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `url` | `url` | No | `url` |  |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |

## `rdp_windows`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `password` | `password` | Yes | `password` |  |
| `username` | `username` | No | `username` |  |
| `port` | `port` | No | `port` |  |
| `allowFileTransfer` | `allowFileTransfer` | No | `allowFileTransfer` |  |

## `rdpldap`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `ldap-hostname` | `ldapHostname` | No | `ldapHostname` |  |
| `ldap-port` | `ldapPort` | No | `ldapPort` |  |
| `ldap-encryption-method` | `ldapEncryptionMethod` | No | `ldapEncryptionMethod` |  |
| `ldap-search-bind-dn` | `ldapSearchBindDn` | No | `ldapSearchBindDn` |  |
| `ldap-search-bind-password` | `ldapSearchBindPassword` | Yes | `ldapSearchBindPassword` |  |
| `ldap-user-base-dn` | `ldapUserBaseDn` | No | `ldapUserBaseDn` |  |
| `ldap-user-name-attribute` | `ldapUserNameAttribute` | No | `ldapUserNameAttribute` |  |

## `redis`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `host` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `tlsEnabled` | `tlsEnabled` | No | `tlsEnabled` |  |
| `tlsSkipVerify` | `tlsSkipVerify` | No | `tlsSkipVerify` |  |
| `isRedisLabs` | `isRedisLabs` | No | `isRedisLabs` |  |
| `crtText` | `tlsCertFile` | Yes | `tlsCertFile` |  |
| `keyText` | `tlsKeyFile` | Yes | `tlsKeyFile` |  |
| `caText` | `tlsCaCert` | No | `tlsCaCert` |  |

## `securetunnels`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |

## `serverlist`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1" |
| `hosts` | `hosts` | No | `hosts` |  |
| `user` | `defaultUser` | No | `defaultUser` |  |
| `sshKey` | `key` | Yes | `key` |  |
| `password` | `password` | Yes | `password` |  |

## `services`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1" |
| `name` | `name` | No | — |  |
| `urls` | `urls` | No | `urls` |  |
| `enableTLS` | `enableTls` | No | `enableTls` |  |

## `slack_webhook`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `webhookURL` | `webhookUrl` | Yes | `webhookUrl` |  |
| `apiToken` | `apiToken` | Yes | `apiToken` |  |

## `snowflake`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `databaseAccount` | `hostname` | No | `hostname` |  |
| `databaseUsername` | `username` | No | `username` |  |
| `databasePassword` | `password` | Yes | `password` |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `warehouse` | `warehouse` | No | `warehouse` |  |
| `schema` | `schema` | No | `schema` |  |
| `clientcert` | `clientcert` | Yes | `clientcert` |  |
| `role` | `role` | No | `role` |  |

## `snowflake_aws_secrets_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `arn` | `arn` | No | `arn` |  |
| `region` | `region` | No | `region` |  |
| `secret_id` | `secretId` | Yes | `secretId` |  |

## `sophos_fw`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `webui_port` | `webuiPort` | No | `webuiPort` |  |
| `login_url` | `loginUrl` | No | `loginUrl` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `splunk`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `tokenID` | `tokenId` | Yes | `tokenId` |  |
| `url` | `url` | No | `url` |  |

## `sql_server`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `databaseName` | `databaseName` | No | `databaseName` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |

## `sqlserver_aws_secrets_manager`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `arn` | `arn` | No | `arn` |  |
| `region` | `region` | No | `region` |  |
| `secret_id` | `secretId` | Yes | `secretId` |  |

## `ssh`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `version` | — | — | — | "1.0" |
| `name` | `name` | No | — |  |
| `username` | `username` | No | `username` |  |
| `usePassword` | `key` | Yes | — |  |
| `password` | `password` | Yes | `password` |  |
| `hostname` | `host` | No | `host` |  |
| `port` | `port` | No | `port` |  |
| `sshKey` | `key` | Yes | `key` |  |

## `syslog`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `hostname` | No | `hostname` |  |
| `port` | `port` | No | `port` |  |
| `protocol` | `protocol` | No | `protocol` |  |
| `tlsEnabled` | `tlsEnabled` | No | `tlsEnabled` |  |
| `caCertificate` | `caCertificate` | No | `caCertificate` |  |
| `clientCertificate` | `clientCertificate` | Yes | `clientCertificate` |  |
| `clientKey` | `clientKey` | Yes | `clientKey` |  |
| `insecureSkipVerify` | `insecureSkipVerify` | No | `insecureSkipVerify` |  |

## `vnc`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hosts` | `hosts` | No | `hosts` |  |
| `port` | `port` | No | `port` |  |
| `password` | `password` | Yes | `password` |  |

## `yugabytedb`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `hostname` | `host` | No | `host` |  |
| `username` | `username` | No | `username` |  |
| `password` | `password` | Yes | `password` |  |
| `sslMode` | `sslMode` | No | `sslMode` |  |
| `rootCert` | `rootCert` | Yes | `rootCert` |  |
| `port` | `port` | No | `port` |  |

## `zerotier`
| Config key | Write arg | Secret? | Read arg | Notes |
|---|---|---:|---|---|
| `name` | `name` | No | — |  |
| `network_id` | `networkId` | No | `networkId` |  |
| `api_token` | `apiToken` | Yes | `apiToken` |  |
| `version` | — | — | — | "1.0" |
