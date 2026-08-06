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

# The same thing on RDS, authenticating with short-lived IAM auth tokens minted
# through a ServiceAccount annotated with eks.amazonaws.com/role-arn, so there
# is no password to store or rotate.
rds_db = adaptive.Resource("my-rds-db",
    name="playground-rds-postgres",
    type="postgres",
    host="mydb.abc123.us-east-1.rds.amazonaws.com",
    port="5432",
    username="iam_user",
    ssl_mode="require",
    use_rds_iam=True,
    use_irsa=True,
    aws_service_account="adaptive-rds-access")

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

pulumi.export("endpointName", endpoint.name)
