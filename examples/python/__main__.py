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

# Example only: this broad sample uses external DB placeholders and cannot be
# run unchanged. The isolated WorkspaceUserRole lab program was run separately.
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

# Workspace user role (local workspace role, independent of IdP roles).
# Requires an existing member. Ensure IdP role sync for this user is disabled
# separately before applying; WorkspaceUserRole does not disable it automatically.
adaptive.WorkspaceUserRole("member-role",
    email="you@example.com",
    role="workspaceapprover")

# Choose which groups from an existing OneLogin integration are included in SSO sync.
# Replace both placeholders with IDs that already exist in the workspace.
adaptive.SsoSync("onelogin-sync",
    provider_id="REPLACE_WITH_EXISTING_SSO_INTEGRATION_ID",
    auto_sync_enabled=True,
    groups=["REPLACE_WITH_IDP_GROUP_ID"])

pulumi.export("endpointName", endpoint.name)
