# Isolated Execution and Job Secrets

Each crawler is launched using `docker run --rm --init`, never as a host process. The job uses a non-root user, a read-only root filesystem, workspace as the only writable bind mount, `/tmp` ephemeral `noexec,nosuid`, capabilities removed, `no-new-privileges`, and limits on CPU, memory, PIDs, and time. The network is `none` by default: enabling egress requires a dedicated Docker network and an operator-managed firewall/allowlist policy.

The disk limit is applied as `--storage-opt size=1g` by default. The Docker host must use a driver with project quotas; if it doesn't support this, the job is rejected instead of running without limits. Docker applies its seccomp profile and AppArmor by default. Custom profiles can be set using `task.sandbox.seccomp` and `task.sandbox.apparmor`.

The worker needs the Docker CLI and the socket. `task.sandbox.workspace_host` must contain the Docker host path corresponding to the worker's workspace; do not use a different container path, as Docker will mount the wrong folder.

Crawler variables do not inherit the process environment. Only `environments` records with a matching `tenant_id`, `project_id`, or `task_id` are injected. Inherited records without scope are not delivered. Each injection generates a record in `secret_access_audits` with the task, tenant/project, and key, but without the value. Injected values ​​are replaced with `[REDACTED]` in the logs.

Minimum production configuration:

```yaml
api:

allow:

origin: https://console.example.com

credentials: "true"

limits:

body_bytes: 1048576
task:

sandbox:

image: goastian/astiango-hub-base:sec-009-011

user: "1000:1000"

network: none

cpus: "1"

memory: 512MB

pids: 128

disk: 1GB

timeout: 30M

workspace_host: /srv/astiango/workspace
```

Do not configure `*` as the origin. API authentication uses Bearer tokens; if a cookie-based session is added, insecure methods must present the matching `astiango_csrf` cookie and `X-CSRF-Token` header.