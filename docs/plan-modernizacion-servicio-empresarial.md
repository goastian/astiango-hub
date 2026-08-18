# Plan de modernización de AstianGO Hub para un servicio empresarial

**Fecha de auditoría:** 17 de agosto de 2026
**Base upstream analizada:** `crawlab-team/crawlab`, copia local inicialmente en `develop` / `ee11cd78eceabb6b4b521b4040d8b5969f4abbef`
**Producto resultante:** `AstianGO Hub`
**Objetivo:** convertir la base actual en un servicio seguro, mantenible, rápido y operable para múltiples empresas sin perder el historial ni la atribución del proyecto original.

### Requisito previo de identidad — aplicado en el árbol de trabajo

Se adoptó el nombre oficial **AstianGO Hub**, el slug `astiango-hub`, el prefijo de configuración `ASTIANGO_`, los módulos `github.com/goastian/astiango-hub/*` y las imágenes `goastian/astiango-hub*`. El contrato completo, los cambios incompatibles y las excepciones de procedencia están en [`docs/rebranding-astiango-hub.md`](rebranding-astiango-hub.md).

El cambio de marca no convierte la base en software propio desde cero ni elimina obligaciones. Se conserva el copyright BSD-3-Clause, la atribución, los disclaimers upstream y el historial Git; AstianGO Hub se declara como fork independiente y no afiliado.

## 1. Decisión ejecutiva

La recomendación es **forkear el monorepo completo `crawlab-team/crawlab` conservando todo su historial**, tomando `develop` en el SHA `ee11cd78...` como punto de evaluación de la futura línea 0.7. No se recomienda reconstruir el producto copiando módulos antiguos ni forkear todos los repositorios de la organización.

La rama `main` está demasiado atrasada para ser una buena base funcional: su último commit consultado es `0485310d` del 9 de octubre de 2024, la última versión estable publicada es `v0.6.3` de julio de 2023 y `develop` contiene la arquitectura 0.7 todavía sin release estable. Por tanto, `develop` aporta más trabajo aprovechable, pero debe tratarse como **código en adquisición**, no como una versión lista para producción.

Antes de incorporar clientes existen bloqueantes de seguridad y confiabilidad:

- secretos de autenticación y cifrado compilados, credenciales iniciales conocidas y cifrado AES-CBC con clave/IV fijo;
- contraseñas almacenadas con MD5, tokens sin expiración y ausencia de autorización fina en operaciones sensibles;
- traversal de rutas en sincronización y operaciones de archivos;
- ejecución deliberada de comandos mediante shell, como root y heredando variables globales, sin aislamiento por cliente o proyecto;
- gRPC sin TLS;
- asignación de tareas no atómica en la práctica, polling cada segundo y una cola en memoria que puede perder tareas al redimensionarse;
- pruebas del backend deshabilitadas en CI, frontend sin pruebas y MCP configurado para pasar aun cuando no hay pruebas;
- imágenes y servicios fuera de soporte: Alpine 3.14, Node.js 20 y MongoDB 5;
- auditoría npm del lockfile con vulnerabilidades críticas y altas;
- despliegue de ejemplo sin persistencia, autenticación de MongoDB, límites de recursos ni alta disponibilidad.

**Conclusión:** se puede conservar una cantidad importante de trabajo, pero el producto debe pasar primero por una etapa de hardening y estabilización de 6 a 10 semanas. Las características comerciales deben construirse después de cerrar los bloqueantes P0.

## 2. Alcance y método

Se inspeccionaron:

- aproximadamente 29.945 líneas Go en `core`, 67.417 líneas Vue/TypeScript/JavaScript en `frontend/astiango-hub-ui`, 3.623 líneas TypeScript en `mcp` y 6.625 líneas de pruebas Go;
- módulos `backend`, `core`, `grpc`, `trace`, `vcs`, `frontend/astiango-hub-ui` y `mcp`;
- Dockerfiles, Compose, workflows de GitHub Actions, configuración, autenticación, scheduler, workers, streams, persistencia, índices, logs y exportaciones;
- los repositorios públicos de la organización `crawlab-team`, sus fechas de actividad, estado archivado, estructura y solapamiento con el monorepo;
- ramas, releases, tags e issues abiertos del repositorio principal;
- versiones y ciclo de vida de Go, Node.js, MongoDB y Alpine;
- `pnpm outdated` y `pnpm audit` sobre los lockfiles del frontend y MCP.

Limitaciones de esta auditoría:

- el entorno no tiene Go instalado, por lo que no fue posible ejecutar `go test`, `go vet`, benchmarks ni `govulncheck`;
- no se instalaron dependencias ni se levantó el clúster; los objetivos de rendimiento son presupuestos que deben validarse mediante carga;
- durante la auditoría los metadatos locales de `.git` fueron reemplazados de forma concurrente por un repositorio vacío `main`. La revisión comenzó y quedó registrada en `develop` / `ee11cd78`; no se intentó reparar `.git` ni modificar el historial;
- los conteos de advisories npm son una fotografía del 17 de agosto de 2026 y pueden cambiar cuando se actualiza la base de advisories.

## 3. Arquitectura actual resumida

| Área | Implementación actual | Riesgo principal |
|---|---|---|
| API/control plane | Go, Gin/Fizz, master único | Autorización insuficiente, secretos fijos, HA inexistente |
| Workers | procesos shell locales, gRPC al master | Sin sandbox por tarea/tenant; heredan entorno del host |
| Scheduler | tareas en MongoDB; workers consultan cada segundo | carga en reposo, asignación duplicada y recuperación ambigua |
| Cola local | canal Go redimensionable | pérdida explícita de tareas durante resize; no durable |
| Estado | MongoDB 5 | EOL; índices recreados destructivamente; retención fija |
| Archivos | filesystem local y sincronización HTTP/gRPC | traversal, falta de durabilidad y cuello de botella del master |
| Logs | archivo por tarea, mutex global, open/write/close | escrituras serializadas y lectura `tail` O(n) |
| Exportaciones | goroutine local + archivos temporales | JSON carga todo en memoria; trabajos se pierden al reiniciar |
| Frontend | Vue 3, Element Plus, Vite | sin pruebas; dependencias antiguas; carga inicial mejorable |
| MCP | servidor TypeScript embebido en el monorepo | dos implementaciones; cero pruebas; filtra prefijo del token al log |
| Entrega | una workflow orientada a construir/publicar imágenes | pruebas omitidas, tags mutables, sin SBOM/firma/promoción |

## 4. Bloqueantes P0 antes de vender el servicio

### 4.1 Seguridad e identidad

- [x] **SEC-001 — Corregir autorización de usuarios.** Aplicado `self-or-admin` a las operaciones del controlador de usuarios, incluido el cambio de contraseña. Los administradores de tenant quedan restringidos a su `tenant_id`; solo el root admin puede gestionar cuentas entre tenants. Se bloquearon los PATCH genéricos de usuarios para evitar escalamiento parcial de privilegios y se añadieron pruebas negativas por rol y tenant.
- [x] **SEC-002 — Reemplazar MD5.** Implementado Argon2id con formato y parámetros versionados (`v=19`, 64 MiB, 3 iteraciones, 2 hilos), comparación en tiempo constante y límites defensivos al procesar hashes almacenados. Los hashes MD5 válidos se migran a Argon2id durante el inicio de sesión. Las cuentas de arranque quedan bloqueadas de la API hasta cambiar su contraseña; el login comunica este requisito y las credenciales inválidas responden `401`.
- [x] **SEC-003 — Gestión real de JWT.** Implementado keyset externo de claves Base64 de mínimo 256 bits, inyectable desde un secret manager o archivo montado, con rotación por `kid`. Se validan estrictamente `HS256`, `iss`, `aud`, `sub`, `iat`, `nbf`, `exp`, `jti` y tipo de token. Se añadieron refresh tokens de uso único y revocación persistente de access/refresh tokens con TTL. El proceso rechaza arrancar si no recibe una configuración JWT segura. Ver [gestión de keysets](jwt-keyset.md).
- [x] **SEC-004 — Eliminar secretos predeterminados.** Las claves de autenticación y cifrado deben inyectarse desde el gestor de secretos y el administrador inicial se provee únicamente para el primer arranque mediante variables de entorno. La rotación está documentada en [migración de secretos y cifrado](secret-rotation-and-encryption-migration.md).
- [x] **SEC-005 — Cifrado autenticado.** AES-256-GCM usa nonce aleatorio, AAD, `kid` y ciphertext versionado; las credenciales de conexiones almacenadas se migran con un comando idempotente. Ver [migración de secretos y cifrado](secret-rotation-and-encryption-migration.md).
- [x] **SEC-006 — Confinar paths.** Se añadió resolución canónica con `EvalSymlinks`, rechazo de rutas absolutas/traversal y escapes por symlink para operaciones de archivos y sincronización. Incluye pruebas unitarias y fuzzing del resolvedor.
- [x] **SEC-007 — Proteger sincronización.** Cada nodo mantiene una credencial aleatoria privada, almacenada como hash en el control plane; sync exige identidad registrada, timestamp, nonce de un solo uso y una tarea activa asignada al recurso. Se aplicó a HTTP y gRPC.
- [ ] **SEC-008 — TLS/mTLS.** TLS para API y mTLS para gRPC entre master y workers, con rotación de certificados y nombres de servicio verificados.
- [x] **SEC-009 — Aislar la ejecución.** Cada tarea se lanza como job Docker efímero no-root con root filesystem de solo lectura, workspace como único mount escribible, `tmpfs` restringido, capabilities eliminadas, `no-new-privileges`, límites CPU/RAM/PID/disco/tiempo y red `none` por defecto. La configuración de seccomp/AppArmor y egress está documentada en [ejecución aislada y secretos](task-sandbox-and-secrets.md).
- [x] **SEC-010 — Segmentar secretos.** Los crawlers ya no heredan `os.Environ()` ni registros globales: solo reciben secretos scoped por tenant/proyecto/tarea. Cada entrega se audita sin persistir el valor y los valores se redactan de la salida de tareas. Ver [ejecución aislada y secretos](task-sandbox-and-secrets.md).
- [x] **SEC-011 — CORS y protección de abuso.** CORS usa allowlist sin wildcard, credenciales solo para orígenes aprobados; se añadieron rate limits por IP/categoría, límite de body, timeouts HTTP, cabeceras seguras y CSRF de doble envío para sesiones basadas en cookies.
- [x] **SEC-012 — Revisar XSS/HTML.** Centralizada la sanitización de todo HTML dinámico con una lista permitida para Markdown, resultados, tooltips/nombres y SVG; se rechazan URLs peligrosas y los enlaces externos usan `noopener noreferrer`. El backend también sanitiza el HTML de las notificaciones Markdown. Los logs se mantienen como texto/Monaco sin interpretación HTML. Se añadió CSP estricta y cabeceras de navegador; se retiraron Clarity, Baidu Analytics y el script inyectado en un SVG para que la instalación funcione sin scripts externos en entornos empresariales/air-gapped.
- [x] **SEC-013 — Proteger MCP.** El token solo se lee desde `ASTIANGO_API_TOKEN_FILE` con permisos privados; no se acepta ni se registra por argumentos o variables de entorno. MCP es solo lectura por defecto y las mutaciones requieren habilitación explícita con un token de permisos mínimos. Las eliminaciones, cancelaciones y deshabilitaciones exigen una confirmación ligada al recurso. Se corrigió la alerta Dependabot #220 y se actualizaron las dependencias MCP/HTTP a versiones sin vulnerabilidades altas conocidas.
- [ ] **SEC-014 — Pipeline de supply chain.** SAST, secret scanning, `govulncheck`, auditoría npm, SBOM CycloneDX/SPDX, escaneo de imágenes, firma Cosign y provenance SLSA.

**Criterio de salida:** cero vulnerabilidades críticas/altas explotables conocidas; suite de autorización negativa aprobada; rotación ensayada; pentest independiente sobre API, archivos, ejecución y aislamiento.

### 4.2 Correctitud de tareas y datos

- [ ] **TASK-001 — Claim atómico.** Sustituir `GetOne + Replace` por `FindOneAndUpdate` con filtro de estado, orden por prioridad/fecha y asignación atómica de `node_id`, `lease_id`, `lease_until` y `attempt`. El `SessionContext` actual no se propaga a las operaciones de modelo, por lo que la transacción no protege el claim.
- [ ] **TASK-002 — Semántica explícita.** Adoptar entrega *at-least-once*, claves de idempotencia y estados con máquina de transición validada. No marcar `cancelled` si no se confirmó que el proceso terminó; distinguir `cancel_requested`, `cancelled` y `cancel_failed`.
- [ ] **TASK-003 — Leases y reconciliación.** Heartbeats renuevan leases; un reconciliador reasigna tareas expiradas con límite de intentos y evita dobles ejecuciones. Completar el TODO de consulta real al worker.
- [ ] **TASK-004 — Cola durable.** Primera opción: mantener MongoDB con claim/lease y change streams; alternativa si la carga lo exige: NATS JetStream, RabbitMQ o Redis Streams detrás de una interfaz. No usar el canal local como fuente de verdad.
- [ ] **TASK-005 — Eliminar pérdida en resize.** No cerrar/reemplazar el canal consumido por workers. Usar un pool con límite dinámico por semáforo o una cola durable. Una reducción de capacidad debe drenar, nunca descartar.
- [ ] **TASK-006 — Idempotencia de schedules.** Lock distribuido por `schedule_id + fire_time`, zona horaria por tenant y deduplicación. Cubrir el issue de ejecuciones múltiples.
- [ ] **TASK-007 — Retención segura.** El índice TTL actual borra tareas por `created_at` a los 30 días sin considerar estado ni política. Reemplazarlo por políticas configurables, legal hold, archivado y purga por lotes.

**Criterio de salida:** cero tareas perdidas o ejecutadas dos veces sin detectar en una prueba de 24 horas con master failover, 100 workers, desconexiones y resize; historial consistente después de reinicios.

### 4.3 Calidad y CI

- [ ] **QA-001 — Restaurar pruebas reales en CI.** Separar unitarias, integración con replica set MongoDB, contrato gRPC/API y E2E. El job actual solo imprime que las pruebas están deshabilitadas.
- [ ] **QA-002 — Traer `crawlab-test`.** Integrarlo como repositorio de pruebas o submódulo fijado por SHA; hoy la documentación local apunta a una carpeta `tests/` inexistente.
- [ ] **QA-003 — Frontend.** Sustituir `vite lint` por ESLint real, añadir typecheck, Vitest + Vue Test Utils y Playwright para flujos críticos.
- [ ] **QA-004 — MCP.** Eliminar `--passWithNoTests`; pruebas de cada tool, schemas, errores, paginación y contrato con una API simulada.
- [ ] **QA-005 — Go.** `gofmt`, `go vet`, `staticcheck`, `golangci-lint`, `go test -race`, fuzz para paths/JWT/filtros y benchmarks del scheduler/logs/export.
- [ ] **QA-006 — Gates.** Ninguna imagen se publica si falla un test, scan o build. Proteger `main`; PR con dos revisiones para seguridad/scheduler; CODEOWNERS.

**Criterio de salida:** pipeline reproducible en PR; cobertura de reglas críticas >= 80% y global inicial >= 60%; ninguna ruta P0 sin test negativo.

## 5. Plan de actualizaciones

Las actualizaciones mayores deben hacerse por lotes pequeños con tests de contrato y rendimiento, no en una sola PR.

| Componente | Actual | Objetivo recomendado | Acción |
|---|---:|---:|---|
| Go | 1.23.7 / builder `golang:1.23` | 1.26.x estable | primero 1.24/1.25 si se requiere bisect; luego 1.26; ejecutar tests/race/vulncheck en cada salto |
| MongoDB | imagen `mongo:5` | 8.0 soportado | ensayo 5→6→7→8, backup/restore, replica set y feature compatibility version |
| Node del build frontend | `node:20-alpine` | Node 24 LTS | Node 20 llegó a EOL en marzo de 2026 |
| Alpine runtime | 3.14 | 3.24 o distroless compatible | 3.14 está sin soporte desde mayo de 2023 |
| pnpm | packageManager 9.9; CI MCP 10.12; Docker instala `latest` | una versión 10.x fijada | usar Corepack y lock consistente; no instalar “latest” |
| Go Mongo driver | v1.17.3 | v2 soportado | adaptar API y añadir compatibilidad/benchmarks |
| gRPC middleware | snapshot de v1 de 2019 | v2 | interceptores de auth, recovery, métricas y tracing |
| `gopsutil` | v3.21 incompatible | v4 soportado | validar Windows/Linux y procesos hijos |
| Frontend Vite | 6.x | 8.x | actualización aislada con bundle diff y E2E |
| Vue/compiler | 3.4.x | 3.5.x estable | mantener versiones exactamente alineadas |
| [x] Lexical | 0.16.x | 0.49.0 | migración completada en rama propia; regresión automatizada y visual del editor |
| TypeScript | 5.5/5.8 | versión soportada por Vue/Vite | no saltar a 7 hasta que el ecosistema objetivo lo soporte |
| Jest | 29 | Vitest actual o Jest 30 | se prefiere Vitest para Vite; primero crear tests |
| Tailwind | 3.x | 4.x | solo después del baseline visual |
| Vuex | 4 | Pinia | migración incremental por store |
| MCP SDK | 1.12.2 | 1.30.x | adaptar APIs y validar todos los tools |
| Zod | 3.x | 4.x | actualizar schemas y errores |
| Axios | frontend 1.7 / MCP 1.6 | 1.19.x | cerrar transitivas vulnerables y validar interceptores |

### 5.1 Acciones inmediatas de dependencias JavaScript

El `pnpm audit` del frontend reportó **3 críticas, 63 altas, 72 moderadas y 11 bajas**; MCP reportó **2 críticas, 43 altas, 32 moderadas y 5 bajas**. Entre las críticas aparecen `form-data`, `tar`, `utils-extend` y `shell-quote`. Los conteos incluyen transitivas y deben confirmarse tras una instalación limpia, pero son suficientes para bloquear release.

- [ ] eliminar `base64-img`/`ajax-request` y `utils-extend` si no son indispensables;
- [ ] sustituir `url-regex` sin parche por validación URL nativa o una librería mantenida;
- [ ] actualizar Axios para obtener `form-data` corregido;
- [ ] actualizar/eliminar cadenas antiguas de `glob`, `minimatch`, `brace-expansion`, `js-yaml`, `tar`, `rollup` y Vite;
- [ ] actualizar `vue-i18n`, Element Plus y sanitización de contenido;
- [ ] deduplicar dependencias que aparecen a la vez en `peerDependencies` y `devDependencies`;
- [ ] mover runtime dependencies reales del frontend a `dependencies`; dejar en peers solo las exigidas a consumidores si realmente se publica como librería;
- [ ] retirar Font Awesome 4/archivos de fuentes duplicados y evaluar tree-shaking de iconos;
- [ ] fijar versiones, regenerar lockfile y repetir audit en una imagen limpia.

### 5.2 Simplificación Go

- [ ] consolidar `backend`, `core`, `grpc`, `trace` y `vcs` en un único módulo Go si no se publicarán por separado;
- [ ] eliminar implementaciones duplicadas: tres paquetes UUID, `pkg/errors`/`juju/errors`/errores estándar y loggers superpuestos;
- [ ] reemplazar `ReneKroon/ttlcache` antiguo; los jobs de exportación no deben depender de cache local;
- [ ] evaluar retirada de `apex/log`, `go-homedir` y otras dependencias inactivas;
- [ ] generar protobuf de forma reproducible y verificar que el código generado esté sincronizado en CI;
- [ ] ejecutar `go mod tidy`, `govulncheck ./...` y una revisión de licencias después de disponer de Go 1.26.

## 6. Optimización de procesos de ejecución y rendimiento

### 6.1 Scheduler y workers

| Mejora | Problema corregido | Objetivo medible |
|---|---|---|
| Change streams/long polling o broker | cada worker consulta nodo, cuenta tareas y hace fetch cada segundo | reducir >=95% las operaciones de MongoDB en reposo |
| Claim atómico con índice compuesto | carrera entre selección y asignación | cero claim duplicado en 1 millón de tareas de prueba |
| Índice `{status,node_id,priority,_id}` y variante por tenant | índices simples no cubren filtro+sort | `explain` sin collection scan; P95 claim <50 ms |
| Contador local + reconciliación | `CountDocuments` por ciclo | no consultar conteo por segundo |
| Backpressure por recursos | modo “unlimited” puede crear un worker por tarea | memoria/CPU dentro del presupuesto; cola visible |
| Retries con backoff+jitter y DLQ | fallos transitorios sin política uniforme | intentos auditables; poison jobs aislados |
| Heartbeats adaptativos | numerosos tickers y pings fijos | tráfico de control proporcional a actividad |

### 6.2 MongoDB y modelos

- [ ] sustituir `Find + Replace` por operaciones parciales atómicas; evitar reemplazar documentos completos y pisar cambios concurrentes;
- [ ] propagar `context.Context` desde HTTP/gRPC hasta MongoDB; prohibir `context.Background()` dentro de requests y cursores;
- [ ] usar timeouts de server selection/socket, pool sizing y command monitoring;
- [ ] cambiar paginación profunda de `$skip` a cursor/keyset por `_id` o `(created_at,_id)`;
- [ ] evitar que `CreateIndexes` haga `DropAll` cuando falta un índice; crear migraciones versionadas, aditivas y reversibles;
- [ ] usar índices parciales para tareas pendientes/leases y TTL solo para datos elegibles;
- [ ] separar series de métricas en time-series collections con retención/downsampling;
- [ ] revisar consultas de estadísticas, eliminar timezone fija `Asia/Shanghai` y cachear agregados por ventana;
- [ ] paginar/batchear la limpieza; no cargar todos los IDs viejos en memoria;
- [ ] introducir repositorios/interfaces testeables en lugar de singletons globales.

### 6.3 Logs, streams y exportaciones

- [ ] reemplazar mutex global y open/close por línea por writers bufferizados por tarea con flush por tamaño/tiempo;
- [ ] hacer tail por offsets/índice o backend de logs; actualmente cuenta y escanea el archivo completo;
- [ ] almacenamiento de logs en objeto (S3/GCS/MinIO) o Loki/OpenSearch, con rotación, compresión, retención y búsqueda;
- [ ] no descartar mensajes silenciosamente cuando la cola de stream está llena; medir backlog y aplicar backpressure/spool durable;
- [ ] evitar una goroutine adicional por cada `Recv`; usar el contexto del stream y límites de vida verificables;
- [ ] JSON streaming con `json.Encoder`/NDJSON; nunca acumular el dataset completo antes de escribir;
- [ ] export jobs durables en DB/cola, con progreso, cancelación, checksum, expiración y descarga desde object storage;
- [ ] CSV con batch size configurable, context cancellation y protección contra CSV formula injection;
- [ ] cerrar cursores y archivos en todos los caminos, registrar errores de `Flush/Close` y limpiar temporales por job.

**Presupuestos iniciales:** tail de log P95 <500 ms para archivos de 10 GB; exportar 10 millones de filas con memoria <100 MB por worker; ingestión sostenida de 10.000 líneas/s por nodo sin bloquear la ejecución.

### 6.4 Frontend

- [ ] lazy-load de rutas/vistas; actualmente casi no hay imports de página dinámicos;
- [ ] eliminar o cargar bajo demanda Three.js del login; evitar copia estática y paquete duplicados;
- [ ] medir bundle con budget CI: JS inicial <500 KB gzip y ningún chunk >250 KB gzip sin excepción;
- [ ] virtualizar tablas/logs/resultados y cancelar requests al cambiar filtros;
- [ ] debounce de búsquedas y paginación por cursor para datasets grandes;
- [ ] cache de consultas con invalidación, deduplicación y estados de error/reintento consistentes;
- [ ] WebSocket/SSE único para eventos de tareas/nodos en vez de polling por vista;
- [ ] retirar fuentes/iconos legacy, preloading selectivo y compresión Brotli;
- [ ] accesibilidad WCAG 2.2 AA, i18n completa y timezone por usuario;
- [ ] métricas reales de Web Vitals sin trackers externos obligatorios.

## 7. Arquitectura objetivo

Mantener inicialmente un **monolito modular** para el control plane y separar solo componentes con una razón operacional clara:

1. **Control API:** organizaciones, usuarios, proyectos, spiders, schedules, políticas y auditoría.
2. **Scheduler/dispatcher HA:** claim durable, leases, retries, fairness y quotas.
3. **Worker agent:** recibe jobs firmados y crea sandboxes/containers efímeros.
4. **Storage adapters:** MongoDB para metadata; object storage para archivos/logs/exportaciones; destinos de resultados mediante interfaces.
5. **Event plane:** change streams inicialmente; broker durable cuando las pruebas demuestren necesidad.
6. **Observabilidad:** OpenTelemetry, métricas Prometheus, logs estructurados y trazas correlacionadas.

No se recomienda dividir prematuramente cada carpeta en microservicios. Primero deben existir contratos, ownership, telemetría y pruebas. Los límites anteriores permiten separar scheduler o workers más adelante sin duplicar modelos.

## 8. Mejoras de operación, entrega y procesos de equipo

### 8.1 Build y despliegue

- [ ] Docker multi-stage reproducible; imágenes base por digest; `pnpm install --frozen-lockfile`; nunca `go mod tidy` dentro del build;
- [ ] runtime no-root, `readOnlyRootFilesystem`, tmpfs y capacidades mínimas;
- [ ] eliminar comandos de depuración `ls` del Dockerfile y artefactos innecesarios;
- [ ] separar imagen de control plane de imágenes de ejecución; no incluir todos los runtimes en el servidor;
- [ ] Compose de desarrollo con volúmenes, health checks y MongoDB autenticado/replica set;
- [ ] Helm chart/Kubernetes con PDB, anti-affinity, autoscaling, NetworkPolicy, secrets externos y rolling updates;
- [ ] artefactos AMD64/ARM64 en releases, no solo bajo ejecución manual;
- [ ] tags inmutables por SemVer+SHA; `latest` solo alias; promoción dev→staging→prod sin recompilar;
- [ ] migraciones pre-deploy, backup verificado y rollback compatible;
- [ ] SBOM, firma, provenance y política de admisión.

### 8.2 Observabilidad y SRE

- [ ] logs JSON con `trace_id`, `task_id`, `tenant_id`, `node_id`, versión y nivel; sin secretos;
- [ ] OpenTelemetry en HTTP, gRPC, MongoDB, scheduler y workers;
- [ ] métricas RED/USE: latencia/error/tráfico API; CPU/memoria/disco; queue depth, wait time, claims, retries, leases, drops y goroutines;
- [ ] health, readiness y startup probes reales; `/health` no puede devolver siempre true;
- [ ] pprof solo en loopback o endpoint administrativo autenticado, nunca `0.0.0.0` sin protección;
- [ ] SLO inicial: API 99,9%, dispatcher 99,95%, P95 API lectura <300 ms, P95 dispatch <1 s;
- [ ] alertas por error budget, no por ruido; runbooks y simulacros de caída de master/Mongo/object storage;
- [ ] backup diario, PITR si aplica, restore mensual; definir RPO <=15 min y RTO <=60 min.

### 8.3 Forma de trabajar

- [ ] ADR para decisiones de cola, tenancy, storage, auth y aislamiento;
- [ ] RFC breve para cambios de contrato y migraciones;
- [ ] trunk-based o ramas cortas, PR pequeñas, Conventional Commits y changelog generado;
- [ ] release train quincenal interno y mensual estable; canary y rollback automático;
- [ ] Definition of Done: tests, telemetría, documentación, migración, rollback y threat model cuando corresponda;
- [ ] CODEOWNERS para auth, scheduler, storage, UI y SDK;
- [ ] triage semanal de issues y vulnerabilidades, SLA por severidad;
- [ ] Renovate/Dependabot agrupado: patches automáticos, minors semanales, majors con RFC;
- [ ] matriz de compatibilidad API/SDK/server y deprecación mínima de dos releases;
- [ ] postmortems sin culpa y registro de acciones con owner/fecha/métrica.

## 9. Nuevas características recomendadas

### 9.1 Imprescindibles para empresas (P1)

- [ ] **Multi-tenancy real:** organización, workspace/proyecto, aislamiento de datos, claves compuestas e índices por `tenant_id`.
- [ ] **RBAC/ABAC:** roles personalizables y permisos por recurso/acción; service accounts.
- [ ] **SSO empresarial:** OIDC/SAML, MFA, SCIM y políticas de sesión.
- [ ] **Audit log inmutable:** quién cambió/ejecutó/leyó qué, antes/después, IP y correlación; exportable a SIEM.
- [ ] **Secret manager:** integración Vault/KMS/cloud secrets, scopes y rotación.
- [ ] **Cuotas y fairness:** concurrencia, CPU/RAM, requests, almacenamiento y presupuesto por tenant/proyecto.
- [ ] **Billing/metering:** task-seconds, CPU/RAM, bytes, resultados y retención; límites y alertas de consumo.
- [ ] **Backups/DR administrables:** estado, archivos, configuraciones y restore por tenant.
- [ ] **Webhooks/eventos:** task lifecycle, alertas, entregas firmadas, retries y DLQ.
- [ ] **API tokens seguros:** scopes, expiración, rotación, último uso y revocación.

### 9.2 Productividad de crawlers (P1/P2)

- [ ] **Versionado de spiders:** commits/releases, diff, rollback, aprobación y promoción entre entornos.
- [ ] **Runtime por proyecto:** imagen/container, Python/Node/Java/Go fijados, lockfile y cache de dependencias por hash.
- [ ] **Workflows/DAG:** dependencias, fan-out/fan-in, parámetros, artefactos y reanudación.
- [ ] **Políticas de retry/timeout:** configurables por error, backoff/jitter, SLA y DLQ.
- [ ] **Entornos dev/staging/prod:** variables, secretos y promociones GitOps.
- [ ] **Debugging:** replay con snapshot, shell/terminal auditada opcional, profiling y comparación entre ejecuciones.
- [ ] **Plantillas mantenidas:** Scrapy, Playwright, Puppeteer, Selenium, Colly y Java; ejemplos verificados en CI.
- [ ] **Calidad de datos:** schema, validación, deduplicación, lineage, sample preview y reglas de aceptación.
- [ ] **Conectores de salida:** S3/GCS/Azure Blob, Kafka, PostgreSQL, MySQL, Elasticsearch/OpenSearch y webhooks.
- [ ] **Proxy/egress policy:** pools, rotación, geografía, salud, coste y allow/deny lists por tenant.

### 9.3 Operación y analítica (P2)

- [ ] dashboard de queue wait, duración, success rate, retries, coste y capacidad;
- [ ] alertas con reglas, silencios, escalamiento y destinos Slack/Teams/PagerDuty/email/webhook;
- [ ] capacity planner y autoscaling de workers por cola/recursos;
- [ ] búsqueda unificada de logs y resultados con correlación de tarea;
- [ ] políticas configurables de retención/archivado/legal hold;
- [ ] mantenimiento de nodos: cordon, drain, labels, pools y afinidad de tareas;
- [ ] ventanas de mantenimiento y calendarios/timezones por organización;
- [ ] status page, historial de incidentes y export de métricas/SIEM.

### 9.4 API, SDK y automatización (P2)

- [ ] OpenAPI versionada como contrato generado desde el servidor y validada en CI;
- [ ] SDKs Go/Python/Node/Java generados cuando sea posible, con capa idiomática mínima y contract tests;
- [ ] CLI única para login, deploy, run, logs, results y administración;
- [ ] Terraform provider o API declarativa para organizaciones, proyectos, spiders y schedules;
- [ ] MCP consolidado, con tools de solo lectura por defecto, scopes y human-in-the-loop para mutaciones;
- [ ] integraciones GitHub/GitLab/Bitbucket con webhooks, commit SHA y deploy previews.

### 9.5 Características posteriores (P3)

- [ ] marketplace de conectores/plugins con paquetes firmados y permisos declarados;
- [ ] scheduler multi-región y data residency;
- [ ] ejecución serverless/burst workers;
- [ ] recomendaciones de optimización basadas en telemetría;
- [ ] generación asistida de crawler con sandbox, revisión humana y políticas de uso;
- [ ] detección de cambios de estructura de páginas y regresión de extracción.

## 10. Qué repositorios forkear y cuáles son redundantes

### 10.1 Forks recomendados

| Repositorio | Decisión | Motivo |
|---|---|---|
| [`crawlab`](https://github.com/crawlab-team/crawlab) | **Fork obligatorio, completo** | fuente canónica; ya contiene backend, core, UI, gRPC, VCS, trace y MCP |
| [`crawlab-test`](https://github.com/crawlab-team/crawlab-test) | **Fork recomendado** | framework de pruebas actual que falta en la copia local; fijar por SHA e incluir Community en CI |
| [`crawlab-docs`](https://github.com/crawlab-team/crawlab-docs) | **Fork recomendado si habrá portal público** | código activo en 2026; mantener versión de docs alineada con releases |
| [`crawlab-python-sdk`](https://github.com/crawlab-team/crawlab-python-sdk) | **Mantener** | SDK dedicado activo y esencial para Scrapy/Python |
| [`crawlab-node-sdk`](https://github.com/crawlab-team/crawlab-node-sdk) | **Mantener si Node está soportado** | SDK dedicado activo en 2026 |
| [`crawlab-go-sdk`](https://github.com/crawlab-team/crawlab-go-sdk) | **Mantener si Go está soportado** | SDK dedicado activo en 2026 |
| [`crawlab-java-sdk`](https://github.com/crawlab-team/crawlab-java-sdk) | **Mantener si Java está soportado** | SDK dedicado activo en 2026 |

Reducir el número inicial de SDKs a los lenguajes con clientes reales. Cada SDK adicional crea una obligación de release, soporte y seguridad. Python debería ser el primero; Node, Go y Java pueden activarse por demanda.

### 10.2 Repositorios redundantes: no forkear como productos separados

| Repositorio(s) | Solapamiento actual | Decisión |
|---|---|---|
| `crawlab-core` (archivado) | `core/` | no forkear; conservar historia desde el monorepo |
| `crawlab-grpc` (archivado) | `grpc/` | no forkear; protobuf y generado viven en el monorepo |
| `crawlab-vcs` | `vcs/` | no forkear; última actividad de código separada en 2023 |
| `go-trace` | `trace/` | absorber como paquete interno o sustituir por OpenTelemetry |
| `crawlab-db` | `core/mongo` y `core/database` | no forkear; implementación separada antigua |
| `crawlab-fs` | `core/fs` y sync actual | no forkear; rescatar solo tests o ideas útiles con atribución |
| `crawlab-log` | `core/task/log` | no forkear; reemplazar por backend de logs mantenido |
| `crawlab-frontend` (archivado) | frontend Vue legado | no forkear |
| `crawlab-ui` (fork upstream) | `frontend/astiango-hub-ui` | no forkear aparte; el monorepo contiene una versión más integrada y ya renombrada |
| `docker-base-images` (archivado) | `docker/base-image` | no forkear; reconstruir imágenes seguras en el monorepo |
| `crawlab-sdk` | SDK antiguo/ambiguo, hoy Go | deprecar; los SDKs por lenguaje lo sustituyen |
| `crawlab-openapi` | contrato API separado y prácticamente vacío | no usar como fuente; generar OpenAPI desde `crawlab` |
| `e2e-tests` | Playwright antiguo | migrar casos útiles a `crawlab-test` y archivar |
| `crawlab-mcp` (Python) | `mcp/` TypeScript en el monorepo | escoger una implementación; preferencia por la integrada TS; migrar tests útiles y deprecar Python |
| `crawlab-plugins`, `plugins`, `public-plugins`, `crawlab-plugin` | tres generaciones de infraestructura de plugins | no forkear hasta definir el nuevo contrato de plugins |
| `plugin-notification` | notificaciones ya presentes en `core/notification` | no forkear; migrar funcionalidad faltante si existe |
| `plugin-dependency` | dependencias ya presentes en `core/dependency` | no forkear |
| `plugin-spider-assistant` | UI/AI actual en el monorepo | no forkear |
| `template-parser`, `autowire` | utilidades antiguas no importadas por el monorepo actual | no forkear; reemplazar o internalizar solo si aparece una necesidad |
| `images`, `resources`, `tutorials`, `crawlab.github.io` | assets/documentación histórica | no mantener como código de producto; copiar solo assets con licencia y procedencia |
| `examples` | ejemplos antiguos referenciados por README | no forkear sin más; crear ejemplos mínimos verificados dentro de docs o un repo nuevo pequeño |
| `crawlab-lite`, `crawlab-next`, `webspot`, `artipub`, `cloudafford`, `scrapy-ai`, `crawlab-ai-sdk`, `crawleval` | productos/experimentos distintos | fuera del alcance del fork principal |
| forks genéricos (`goseaweedfs`, `fizz`, `mcp-go`, `bm25`, etc.) | dependencias upstream o experimentos | depender de upstream mantenido o fijar SHA; no heredar su mantenimiento sin necesidad |

### 10.3 Cómo evitar perder trabajo

1. Crear un fork de GitHub de `crawlab-team/crawlab`; no crear un repositorio vacío ni copiar carpetas.
2. Configurar `origin` como el fork propio y `upstream` como `https://github.com/crawlab-team/crawlab.git`.
3. Etiquetar el punto base exacto: `upstream-snapshot-2025-12-03-ee11cd78`.
4. Mantener una rama espejo sin cambios de `upstream/develop` y una rama propia protegida para el producto.
5. Integrar upstream mediante PRs periódicas y selectivas; registrar conflictos y decisiones en `docs/upstream-sync.md`.
6. No hacer squash de toda la historia inicial. Usar commits normales para mantener `git blame`, autoría y capacidad de comparar.
7. Antes de archivar un repo redundante, comparar licencias, tags, tests y commits no presentes; importar únicamente cambios faltantes con `cherry-pick -x` o subtree cuando corresponda.
8. Preservar `LICENSE`, notices y autoría. El monorepo es BSD-3-Clause, mientras algunos componentes históricos usan Apache-2.0 o MIT.
9. Crear un inventario firmado de SHAs base de `crawlab`, tests, docs y SDKs.
10. Hacer el primer release propio solo después de reproducir build, backup/restore, migración y rollback.

## 11. Hoja de ruta propuesta

### Fase 0 — Contención y baseline (semanas 1-2)

- congelar releases públicos;
- restaurar `.git` desde una copia/fork verificada y registrar SHA base;
- crear threat model y reproducir los issues críticos;
- corregir secretos, passwords, autorización y paths;
- actualizar Alpine/Node/Mongo en un entorno de prueba;
- instalar Go 1.26 y obtener baseline de build/tests/vulnerabilidades;
- integrar CI mínima y `crawlab-test`;
- capturar benchmarks de idle, dispatch, logs, export y frontend.

**Gate:** sin críticos reproducibles; build limpio y firmado; tests P0 verdes.

### Fase 1 — Confiabilidad del núcleo (semanas 3-6)

- claim atómico, leases, máquina de estados e idempotencia;
- eliminar pérdida en worker pool y corregir cancelación/reconciliación;
- migraciones de índices/retención;
- logs bufferizados y JSON streaming;
- Compose productivo de referencia y observabilidad básica;
- contrato OpenAPI y primera SDK Python compatible.

**Gate:** prueba chaos de 24 h, cero pérdida/duplicados no detectados, restore probado.

### Fase 2 — Rendimiento y HA (semanas 7-10)

- change streams/long polling, reducción de polling y keyset pagination;
- control plane replicable con leader election/locks donde se requiera;
- object storage y jobs de export durable;
- métricas/trazas/SLO, autoscaling inicial;
- optimización de bundle y vistas de alto volumen.

**Gate:** objetivos P95 y reducción >=95% de carga idle; canary/rollback automático.

### Fase 3 — MVP empresarial (semanas 11-16)

- tenancy, RBAC, OIDC, audit log, secrets, quotas y webhooks;
- runtimes aislados por proyecto y versionado de spiders;
- portal operativo, backups por tenant y documentación;
- billing/metering mínimo y soporte/runbooks.

**Gate:** pentest externo, revisión de aislamiento multi-tenant, piloto con 2-3 empresas y SLO durante 30 días.

### Fase 4 — Expansión (posterior)

- SDKs adicionales, DAGs, conectores, marketplace firmado, multi-región y capacidades AI seguras;
- priorizar únicamente con evidencia de clientes y métricas de adopción.

## 12. KPIs para comprobar que el plan funciona

| Dimensión | KPI |
|---|---|
| Seguridad | 0 críticos/altos explotables; 100% endpoints con matriz de autorización; rotación ensayada |
| Confiabilidad | pérdida de tareas = 0; duplicados no detectados = 0; recuperación de lease <2 min |
| Rendimiento | P95 dispatch <1 s; P95 API lectura <300 ms; carga Mongo idle -95% |
| Logs/export | tail P95 <500 ms; export 10 M filas <100 MB RAM/worker |
| Frontend | LCP P75 <2,5 s; INP P75 <200 ms; JS inicial <500 KB gzip |
| Calidad | gates CI 100%; core crítico >=80% cobertura; flake rate <1% |
| Operación | disponibilidad API 99,9%; dispatcher 99,95%; RPO <=15 min; RTO <=60 min |
| Entrega | lead time <2 días; rollback <10 min; releases sin recompilar entre ambientes |
| Producto | activación, tareas exitosas/tenant, coste por task-hour y retención de clientes |

## 13. Fuentes consultadas

- [Repositorio principal Crawlab](https://github.com/crawlab-team/crawlab)
- [Repositorios de la organización crawlab-team](https://github.com/orgs/crawlab-team/repositories?type=all)
- [Releases de Crawlab](https://github.com/crawlab-team/crawlab/releases)
- [Issue 1622: secreto JWT fijo](https://github.com/crawlab-team/crawlab/issues/1622)
- [Issue 1623: autorización de cambio de password](https://github.com/crawlab-team/crawlab/issues/1623)
- [Issue 1619: traversal de rutas](https://github.com/crawlab-team/crawlab/issues/1619)
- [Issue 1597: carga MongoDB en reposo](https://github.com/crawlab-team/crawlab/issues/1597)
- [Issue 1488: schedules duplicados](https://github.com/crawlab-team/crawlab/issues/1488)
- [Issue 1421: aislamiento de entornos](https://github.com/crawlab-team/crawlab/issues/1421)
- [Issue 1535: rotación de logs](https://github.com/crawlab-team/crawlab/issues/1535)
- [Historial y política de releases de Go](https://go.dev/doc/devel/release)
- [Ciclo de vida de Node.js](https://nodejs.org/en/about/previous-releases)
- [Ciclo de vida de MongoDB](https://www.mongodb.com/legal/support-policy/lifecycles)
- [Ciclo de vida de Alpine Linux](https://www.alpinelinux.org/releases/)
- [Advisory `form-data`](https://github.com/advisories/GHSA-fjxv-7rqg-78g4)
- [Advisory `tar`](https://github.com/advisories/GHSA-23hp-3jrh-7fpw)
- [Advisory `shell-quote`](https://github.com/advisories/GHSA-w7jw-789q-3m8p)
