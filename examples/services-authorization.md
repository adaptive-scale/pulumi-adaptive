# Services authorization

Services authorizations use the existing `Authorization` resource with
`resourceType: "servicelist"`. The backend must support Services authorization;
existing resources and SDK argument names are unchanged.

```typescript
import * as adaptive from "@adaptive-scale/pulumi-adaptive";

const servicesRead = new adaptive.Authorization("services-read", {
    name: "services-read",
    resourceType: "servicelist",
    permissions: `allow:
  - operation_type: GET
    route: /api/*
deny:
  - operation_type: "*"
    route: /api/admin/*
`,
    description: "Read API routes except administrative routes",
});
```

Pass `servicesRead.name` to the Services endpoint's `authorization` argument.
In Python, use `resource_type="servicelist"`; in Go, use
`ResourceType: pulumi.String("servicelist")`; in .NET, use
`ResourceType = "servicelist"`. All SDKs accept the same permissions string.

Rules contain an HTTP method (`operation_type`) and a path (`route`). A quoted
`"*"` matches any method; a trailing `*` matches a path prefix. Deny rules take
precedence. At least one allow or deny rule is required. Unknown fields and
`payload_regex` are rejected. Updates containing invalid Services rules fail
without replacing the saved policy.
