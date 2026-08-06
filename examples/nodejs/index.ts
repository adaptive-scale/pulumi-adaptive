// Example Pulumi program (TypeScript) using the Adaptive provider. This
// mirrors the Terraform playground: a Postgres resource, an endpoint that
// references it, an authorization, and a group.
//
// Run with:
//
//	export ADAPTIVE_SVC_TOKEN=...        # or ADAPTIVE_URL for a custom workspace
//	npm install
//	pulumi stack init dev
//	pulumi up

import * as adaptive from "@adaptive-scale/pulumi-adaptive";

// A connection to a Postgres database.
const db = new adaptive.Resource("my-db", {
    name: "playground-postgres",
    type: "postgres",
    host: "db.example.com",
    port: "5432",
    username: "admin",
    password: "not-a-real-password",
    sslMode: "require",
});

// The same thing on RDS, authenticating with short-lived IAM auth tokens minted
// through a ServiceAccount annotated with eks.amazonaws.com/role-arn, so there
// is no password to store or rotate.
const rdsDb = new adaptive.Resource("my-rds-db", {
    name: "playground-rds-postgres",
    type: "postgres",
    host: "mydb.abc123.us-east-1.rds.amazonaws.com",
    port: "5432",
    username: "iam_user",
    sslMode: "require",
    useRdsIam: true,
    useIrsa: true,
    awsServiceAccount: "adaptive-rds-access",
});

// A time-limited access endpoint. Referencing db.name makes Pulumi create
// the resource before the endpoint (the dependency graph).
const endpoint = new adaptive.Endpoint("my-db-access", {
    name: "playground-db-access",
    resource: db.name,
    ttl: "8h",
    users: ["you@example.com"],
});

// A read-only permission policy. For postgres, permissions must be
// structured YAML (a bare "SELECT" is rejected server-side).
const readOnly = new adaptive.Authorization("read-only", {
    name: "playground-readonly",
    resourceType: "postgres",
    description: "Read-only access for the playground",
    permissions: `allow:
  - database: production
    privileges:
      - SELECT
    objects:
      - ALL
`,
});

// A group bundling users and the endpoint above.
const developers = new adaptive.Group("developers", {
    name: "playground-developers",
    members: ["you@example.com"],
    endpoints: [endpoint.name],
});

export const endpointName = endpoint.name;
