# list auth methods (for OIDC accessor discovery)
path "sys/auth"   { capabilities = ["read"] }
path "sys/auth/*" { capabilities = ["read"] }

# manage policies for per-NS access (create/update/list)
# manage policies (ACL API)
path "sys/policies/acl"   { capabilities = ["list","read"] }
path "sys/policies/acl/*" { capabilities = ["create","update","delete","list","read"] }

# enable/tune kv engines only (mount management)
path "sys/mounts"      { capabilities = ["list","read"] } 
path "sys/mounts/*"    { capabilities = ["create","update","delete","read","sudo"] }

# manage kubernetes auth roles
path "auth/k8s-kind/role/*" { capabilities = ["create","update","delete","list","read"] }

# manage identity groups & aliases (no secret reads)
path "identity/group"        { capabilities = ["create","update","list","read"] }
path "identity/group/name/*" { capabilities = ["read"] }
path "identity/group/id/*"   { capabilities = ["create","update","list","read"] }
path "identity/group-alias"  { capabilities = ["create","update","list","read"] }

# allow Vault CLI preflight (detect mount + kv version)
path "sys/internal/ui/mounts"   { capabilities = ["read"] }
path "sys/internal/ui/mounts/*" { capabilities = ["read"] }
