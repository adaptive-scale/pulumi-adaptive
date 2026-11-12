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

const databaseHost = new adaptive.Variable("database-host", {
    name: "database_host",
    value: "db.example.com",
    labels: ["production", "database"],
});

// A connection to a Postgres database.
const db = new adaptive.Resource("my-db", {
    name: "playground-postgres",
    type: "postgres",
    host: "{{database_host}}",
    port: "5432",
    username: "admin",
    password: "not-a-real-password",
    sslMode: "require",
}, { dependsOn: [databaseHost] });

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
