"""Example Pulumi program (Python) using the Adaptive provider. This mirrors
the Terraform playground: a Postgres resource, an endpoint that references it,
an authorization, and a group.

Run with:

    export ADAPTIVE_SVC_TOKEN=...        # or ADAPTIVE_URL for a custom workspace
    pulumi stack init dev
    pulumi up
"""

import pulumi
import pulumi_adaptive as adaptive

# A connection to a Postgres database.
db = adaptive.Resource("my-db",
    name="playground-postgres",
    type="postgres",
    host="db.example.com",
    port="5432",
    username="admin",
    password="not-a-real-password",
    ssl_mode="require")

# A time-limited access endpoint. Referencing db.name makes Pulumi create the
# resource before the endpoint (the dependency graph).
endpoint = adaptive.Endpoint("my-db-access",
    name="playground-db-access",
    resource=db.name,
    ttl="8h",
    users=["you@example.com"])

# A read-only permission policy. For postgres, permissions must be structured
# YAML (a bare "SELECT" is rejected server-side).
read_only = adaptive.Authorization("read-only",
    name="playground-readonly",
    resource_type="postgres",
    description="Read-only access for the playground",
    permissions="""allow:
  - database: production
    privileges:
      - SELECT
    objects:
      - ALL
""")

# A group bundling users and the endpoint above.
developers = adaptive.Group("developers",
    name="playground-developers",
    members=["you@example.com"],
    endpoints=[endpoint.name])

# Workspace settings (singleton). Only the fields set here are managed.
adaptive.WorkspaceSettings("workspace",
    mfa_enabled=True,
    timezone="Europe/Stockholm",
    max_jit_request_duration="8h")

# Choose which IdP groups are included in SSO sync. provider_id is the id of an
# existing SSO integration; group order does not matter.
adaptive.SsoSync("okta-sync",
    provider_id="REPLACE_WITH_SSO_INTEGRATION_ID",
    auto_sync_enabled=True,
    groups=["00g1abc", "00g2def"])

pulumi.export("endpointName", endpoint.name)
