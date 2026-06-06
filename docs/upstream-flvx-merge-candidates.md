# Sagit-chu/flvx Merge Candidate List

Source repository: `D:\Data\RSSQ\flvx` (`Sagit-chu/flvx`, commit `6e249a5`)

Target repository: `D:\Data\RSSQ\flvxt2` (`iKeilo/FLVXR2`, commit `e6633a49`)

Comparison summary:

- Common files: 805
- Same content: 597
- Same path but changed: 208
- Upstream-only files: 196
- Target-only files: 202

This list is scoped to files that matter for runtime behavior, security, installation, backend APIs, agent behavior, and frontend product surfaces. Generated docs assets and historical planning notes are intentionally excluded unless they affect implementation.

## Implementation Status

Started in this branch:

- Panel Peering provider capability handshake now returns `providerType`, `protocolVersion`, feature flags, runtime modes, and explicit WG/TLS support flags.
- Imported legacy providers without capability fields are normalized as `sagit-chu-compatible` baseline providers.
- Remote shared node metadata is persisted in `remote_config` and shown in the Panel Sharing remote-usage cards.
- Remote shared nodes are blocked from local-only node deployment and WG Path creation/update.
- The WG Path frontend selector filters out remote shared nodes.
- Upstream SSRF-oriented remote address safety helpers have been migrated into `security_utils.go`.
- Panel Peering management endpoints are explicitly admin-only: share list/create/update/delete/reset, remote usage list, and remote node import.
- Remote node import now validates URL shape, limits schemes to `http`/`https`, and rejects loopback/private/link-local/unspecified remote hosts.
- Panel Sharing frontend now shows an unavailable state for non-admin users instead of calling protected APIs.
- Compatibility and permission tests cover baseline provider normalization, capability parsing, remote import admin gating, and restricted URL rejection.

Still pending:

- Importing the remaining upstream safety/test files and adapting their fixtures to FLVX commercial auth, HttpOnly cookie auth, WG, TLS templates, and shop schema.
- Optional deeper Sagit-chu share-link parsing if their UI exposes non-URL token formats beyond the current remote URL + token import flow.
- UI-level capability gating for any future provider that advertises extended WG/TLS support.
- Optional admin allowlist for private Panel Peering targets if private-network panel sharing is required in controlled deployments.

## Directly Migratable

These files are absent from `flvxt2` or are mostly additive test/utility files. They can usually be copied first, then compiled to reveal any small dependency gaps.

### Backend Tests And Safety Coverage

- `go-backend/internal/http/handler/config_access_test.go`
- `go-backend/internal/http/handler/flow_upload_batch_test.go`
- `go-backend/internal/http/handler/forward_proxy_protocol_test.go`
- `go-backend/internal/http/handler/nftables_runtime_test.go`
- `go-backend/internal/http/handler/system_upgrade_test.go`
- `go-backend/internal/http/handler/tunnel_best_exit_test.go`
- `go-backend/internal/http/handler/tunnel_probe_target_api_test.go`
- `go-backend/internal/http/handler/tunnel_probe_target_test.go`
- `go-backend/internal/http/handler/tunnel_quality_prober_test.go`
- `go-backend/internal/http/handler/upgrade_test.go`
- `go-backend/internal/http/middleware/auth_test.go`
- `go-backend/internal/store/repo/config_policy_test.go`
- `go-backend/internal/store/repo/repository_auth_test.go`
- `go-backend/internal/store/repo/repository_backup_test.go`
- `go-backend/internal/store/repo/repository_flow_batch_test.go`
- `go-backend/internal/store/repo/repository_forward_proxy_protocol_test.go`
- `go-backend/internal/store/repo/repository_nftables_test.go`
- `go-backend/internal/store/repo/repository_storage_test.go`
- `go-backend/internal/ws/server_test.go`
- `go-backend/tests/contract/flow_upload_batch_contract_test.go`
- `go-backend/tests/contract/forward_local_remote_addr_contract_test.go`
- `go-backend/tests/contract/max_conn_limit_contract_test.go`
- `go-backend/tests/contract/per_ip_speed_limit_contract_test.go`
- `go-backend/tests/contract/storage_contract_test.go`
- `go-backend/tests/contract/user_list_max_conn_contract_test.go`

Notes:

- Some tests may need fixture updates because `flvxt2` has commercial license checks, HttpOnly cookie auth, WG mode, TLS templates, and shop-related schema changes.
- These are still valuable because they define expected behavior and catch regressions.

### Backend Utility Modules

- `go-backend/internal/http/handler/security_utils.go` - migrated in this branch
- `go-backend/internal/security/password.go`
- `go-backend/internal/security/password_test.go`
- `go-backend/internal/auth/user_state.go`

Notes:

- These are low-risk if symbols do not conflict.
- `password.go` should be checked against current login/register logic before replacing any existing hash behavior.

### Repository Helpers

- `go-backend/internal/store/repo/config_policy.go`
- `go-backend/internal/store/repo/repository_auth.go`
- `go-backend/internal/store/repo/repository_backup_test.go`
- `go-backend/internal/store/repo/repository_pool_test.go`
- `go-backend/internal/store/repo/repository_storage.go`

Notes:

- Additive repository methods are good candidates.
- Do not overwrite `repository.go`, `repository_mutations.go`, or `model.go` blindly because `flvxt2` has extra commercial, deployment, TLS, WG, and payment tables.

### GOST/Agent Tests

- `go-gost/x/config/parsing/service/composite_limiter_test.go`
- `go-gost/x/config/persist_test.go`
- `go-gost/x/handler/forward/local/proxy_protocol_test.go`
- `go-gost/x/limiter/conn/conn_test.go`
- `go-gost/x/limiter/traffic/traffic_test.go`
- `go-gost/x/listener/udp/listener_test.go`
- `go-gost/x/registry/chain_test.go`
- `go-gost/x/service/global_traffic_manager_test.go`
- `go-gost/x/socket/chain_test.go`
- `go-gost/x/socket/limiter_test.go`
- `go-gost/x/socket/service_test.go`

Notes:

- Prefer importing tests first, then selectively port production changes only where tests fail.

## Needs Adaptation

These files contain useful upstream behavior, but they overlap with `flvxt2` business changes or runtime design. They must be merged manually.

### nftables Runtime

- `go-backend/internal/http/handler/nftables_runtime.go`
- `go-backend/internal/runtime/nftables/manager.go`
- `go-backend/internal/runtime/nftables/parser.go`
- `go-backend/internal/runtime/nftables/renderer.go`
- `go-backend/internal/runtime/nftables/runner.go`
- `go-backend/internal/runtime/nftables/types.go`
- `go-backend/internal/store/repo/repository_nftables.go`

Why adaptation is needed:

- Upstream uses a backend-managed nftables runtime path with SSH reconciliation.
- `flvxt2` already has agent/socket based nftables and WG Path deployment logic in `go-gost/x/socket/nftables_handler.go`, `go-gost/x/socket/wireguard_handler.go`, and `go-backend/internal/http/handler/wg_path.go`.
- A direct copy could create two independent nftables control planes.

Recommended merge strategy:

- Extract upstream parser/renderer/runner as a reusable runtime package.
- Wire it behind the existing `flvxt2` node deployment/diagnosis flow.
- Keep WG Path authority in `wg_path.go`.
- Add an explicit runtime mode boundary: `agent_socket`, `backend_ssh`, `wg_path`.

### Tunnel Probe And Best Exit

- `go-backend/internal/http/handler/tunnel_probe_target.go`
- `go-backend/internal/http/handler/tunnel_best_exit.go`
- `go-backend/internal/http/handler/tunnel_best_exit_display.go`
- `go-backend/internal/http/handler/tunnel_quality_prober.go`
- `go-backend/internal/store/repo/repository_tunnel_quality.go`
- `vite-frontend/src/pages/tunnel.tsx`
- `vite-frontend/src/pages/tunnel/diagnosis.ts`
- `vite-frontend/src/pages/tunnel/form.ts`

Why adaptation is needed:

- `flvxt2` has WG Path, commercial gating, revised UI, and tunnel grouping changes.
- Upstream quality probing is valuable for NAT/WG fallback decisions, but UI and schema must fit the current tunnel manager.

Recommended merge strategy:

- Port backend scoring/probe logic first.
- Expose it to WG Path as optional diagnostics.
- Merge frontend display only after API payloads stabilize.

### Forward Management

- `go-backend/internal/http/handler/mutations.go`
- `go-backend/internal/http/handler/user_quota.go`
- `go-backend/internal/store/repo/repository_mutations.go`
- `go-backend/internal/store/repo/repository_user_quota.go`
- `vite-frontend/src/pages/forward.tsx`
- `vite-frontend/src/pages/forward/batch-actions.ts`
- `vite-frontend/src/pages/forward/order.ts`
- `vite-frontend/src/api/types.ts`

Why adaptation is needed:

- Upstream supports `agent | nftables`.
- `flvxt2` supports `gost | nftables | wg_path` with admin-only WG restrictions.
- Direct copy would likely remove WG Path rule creation and ordinary-user filtering.

Recommended merge strategy:

- Diff only the specific upstream fixes: local/remote address validation, proxy protocol, stale runtime cleanup, port occupancy, failure detail handling.
- Preserve `wg_path` branches and admin checks.

### Panel Peering / 面板共享

- `go-backend/internal/http/handler/federation.go`
- `go-backend/internal/http/client/federation.go`
- `go-backend/internal/store/repo/repository_federation.go`
- `go-backend/internal/store/model/model.go` peer-sharing structs
- `go-backend/internal/http/handler/mutations.go` federation runtime binding sections
- `go-backend/internal/http/handler/flow_policy.go`
- `go-backend/internal/http/handler/flow_policy_federation_test.go`
- `go-backend/tests/contract/federation_dual_panel_contract_test.go`
- `go-backend/tests/contract/federation_forward_flow_linkage_contract_test.go`
- `vite-frontend/src/pages/panel-sharing.tsx`
- `vite-frontend/src/api/index.ts` federation API section
- `vite-frontend/src/layouts/admin.tsx` navigation entry

Why adaptation is needed:

- Both repositories already implement a federation-style panel sharing model, but `flvxt2` has extra commercial licensing, HttpOnly Cookie browser auth, WG Path, TLS/inbound deployment, and product visibility rules.
- Upstream `Sagit-chu/flvx` currently provides the useful compatibility target because it exposes the same family of endpoints:
  - `/api/v1/federation/share/list`
  - `/api/v1/federation/share/create`
  - `/api/v1/federation/share/update`
  - `/api/v1/federation/share/delete`
  - `/api/v1/federation/share/reset-flow`
  - `/api/v1/federation/share/remote-usage/list`
  - `/api/v1/federation/connect`
  - `/api/v1/federation/tunnel/create`
  - `/api/v1/federation/runtime/reserve-port`
  - `/api/v1/federation/runtime/apply-role`
  - `/api/v1/federation/runtime/release-role`
  - `/api/v1/federation/runtime/diagnose`
  - `/api/v1/federation/runtime/command`
  - `/api/v1/federation/node/import`
- Direct replacement is unsafe because `flvxt2` has already moved more logic into commercial boundaries and has additional node metadata such as deployment identity, TLS templates, and WG-related behavior.

Compatibility behavior to preserve or add:

1. Provider side creates a share:
   - Admin chooses local node, name, bandwidth limit, flow limit, port range, expiry, allowed domains, and allowed IPs.
   - Backend creates or updates `peer_share`.
   - Backend returns a share URI containing at least `remote_url` and `token`.
   - Share token must stay compatible with Sagit-chu import flow.

2. Consumer side imports a share:
   - User pastes a Sagit-chu-compatible share URI or manually enters `remote_url + token`.
   - `flvxt2` calls provider `/api/v1/federation/connect`.
   - If response is old/upstream format, normalize it into the current `flvxt2` remote node schema.
   - Imported node is stored as a remote/federated node, not as a normal locally managed deployment node.
   - Imported node must not expose local-only actions such as node deploy, TLS deployment, WG Path install, or core config push.

3. Runtime reservation:
   - When a tunnel uses a remote federated node, `flvxt2` calls `/runtime/reserve-port` on the provider.
   - Request must include local domain, resource key, tunnel role, desired protocol, strategy, and requested port when available.
   - If provider is Sagit-chu and does not know newer fields, omit unknown fields or tolerate ignored fields.
   - Store the returned allocation in `federation_tunnel_binding`.

4. Runtime apply:
   - After reservation, call `/runtime/apply-role`.
   - Role should be one of `entry`, `middle`, `exit`, or `forward`.
   - For Sagit-chu compatibility, use GOST/nftables-compatible tunnel runtime payloads only.
   - Do not send `wg_path`, S-UI inbound templates, Reality TLS templates, or sing-box-only configs to an upstream Sagit-chu provider unless a capability handshake says it supports them.

5. Runtime release:
   - On tunnel update/delete/failure rollback, call `/runtime/release-role`.
   - Release must be best effort but must also mark local `federation_tunnel_binding` status inactive.
   - Release by `resource_key` first, fallback to `binding_id` or allocated port for older provider behavior.

6. Flow accounting:
   - Keep local display flow compatible with upstream `peer_share.current_flow`.
   - Merge flow from:
     - provider-side share usage
     - local tunnel/forward usage linked through `federation_tunnel_binding`
     - remote usage list from `/share/remote-usage/list`
   - When remote provider lacks detailed per-runtime usage, fall back to share-level flow split by active bindings, matching current upstream contract tests.

7. Diagnostics:
   - For remote federated nodes, tunnel/forward diagnosis should proxy to `/runtime/diagnose`.
   - UI should display the same diagnosis panel used by local tunnel/forward diagnosis.
   - If provider returns unsupported command, show a compatibility warning instead of marking the local node broken.

8. Capability handshake:
   - Add optional capability detection on `/federation/connect`.
   - Expected new optional fields:
     - `protocolVersion`
     - `features`
     - `runtimeModes`
     - `supportsNftables`
     - `supportsWGPath`
     - `supportsTLSInbound`
   - If missing, assume Sagit-chu-compatible baseline:
     - `features = ["gost_tunnel", "port_forward", "runtime_reserve", "runtime_apply", "runtime_release", "runtime_diagnose"]`
     - `supportsWGPath = false`
     - `supportsTLSInbound = false`

9. Permission model:
   - Share creation and share management should remain admin-only in `flvxt2`.
   - Imported federated nodes may be assigned to groups/users according to existing group permission rules.
   - Ordinary users must not be able to create provider shares or import arbitrary remote shares unless explicitly enabled by commercial policy.
   - WG Path remains admin-only even if a remote panel advertises WG support.

10. Failure behavior:
   - If remote reservation succeeds but local tunnel creation fails, release the remote reservation immediately.
   - If remote apply fails after reservation, release and surface the provider error.
   - If provider is unreachable during update, do not delete old working local binding until rollback is decided.

Recommended merge strategy:

- Treat Sagit-chu compatibility as a protocol adapter, not as a wholesale file copy.
- Keep `flvxt2` data model as the authoritative local model.
- Add a `federation provider capability` normalization layer in `go-backend/internal/http/client/federation.go`.
- Add contract tests with two simulated providers:
  - baseline Sagit-chu-compatible provider with no capability fields
  - flvxt2 provider with extended capability fields
- Keep UI labels as “面板共享 / Panel Peering”, but show imported provider type:
  - `FLVX compatible`
  - `Sagit-chu compatible`
  - `Unknown legacy`
- Do not expose WG/TLS deployment buttons on imported Sagit-chu remote nodes.

### Auth And Session

- `go-backend/internal/http/middleware/auth.go`
- `go-backend/internal/http/middleware/cors.go`
- `go-backend/internal/auth/jwt.go`
- `vite-frontend/src/api/index.ts`
- `vite-frontend/src/api/network.ts`
- `vite-frontend/src/utils/auth.ts`
- `vite-frontend/src/utils/jwt.ts`
- `vite-frontend/src/utils/logout.ts`
- `vite-frontend/src/utils/session.ts`

Why adaptation is needed:

- `flvxt2` now uses HttpOnly cookie as the main browser auth path.
- Upstream still expects token-style flows in several places.

Recommended merge strategy:

- Only import tests or narrowly scoped validation improvements.
- Do not replace cookie/session behavior.

### WebSocket And Agent Runtime

- `go-backend/internal/ws/server.go`
- `go-gost/x/socket/websocket_reporter.go`
- `go-gost/x/socket/websocket_reporter_test.go`
- `go-gost/x/socket/service.go`
- `go-gost/x/socket/chain.go`
- `go-gost/x/socket/limiter.go`

Why adaptation is needed:

- `flvxt2` added `ApplyCoreConfig`, WireGuard config application, cookie fallback for panel WebSocket auth, and node deployment commands.
- Upstream may contain stability fixes, but direct replacement could reintroduce `未知命令类型: ApplyCoreConfig`.

Recommended merge strategy:

- Compare command type registry and message handlers first.
- Keep all `ApplyCoreConfig`, WG, TLS deployment, and node identity handling.
- Cherry-pick reconnect, limiter, status, and telemetry fixes.

### go-gost Network Core

- `go-gost/config.go`
- `go-gost/main.go`
- `go-gost/program.go`
- `go-gost/x/config/config.go`
- `go-gost/x/config/parsing/service/parse.go`
- `go-gost/x/config/persist.go`
- `go-gost/x/handler/forward/local/handler.go`
- `go-gost/x/handler/forward/local/metadata.go`
- `go-gost/x/internal/net/udp/listener.go`
- `go-gost/x/listener/udp/listener.go`
- `go-gost/x/listener/udp/metadata.go`
- `go-gost/x/service/global_traffic_manager.go`
- `go-gost/x/service/service.go`

Why adaptation is needed:

- These are central traffic-path files.
- Upstream has useful UDP, retry, proxy protocol, and traffic manager changes.
- `flvxt2` has agent-specific reporting, WG, nftables socket handlers, and deployment assumptions.

Recommended merge strategy:

- Port tests first.
- Then port one protocol/runtime area at a time with performance checks.

### Frontend UI And Layout

- `vite-frontend/src/layouts/admin.tsx`
- `vite-frontend/src/layouts/default.tsx`
- `vite-frontend/src/pages/dashboard.tsx`
- `vite-frontend/src/pages/node.tsx`
- `vite-frontend/src/pages/tunnel.tsx`
- `vite-frontend/src/pages/forward.tsx`
- `vite-frontend/src/pages/config.tsx`
- `vite-frontend/src/components/navbar.tsx`
- `vite-frontend/src/components/version-footer.tsx`
- `vite-frontend/src/styles/globals.css`
- `vite-frontend/src/styles/tailwind-theme.pcss`

Why adaptation is needed:

- `flvxt2` has custom modern glass UI, commercial visibility rules, TLS inside node menus, WG controls, shop tabs, and branding restrictions.
- Upstream UI changes can be useful, but direct replacement would regress current product behavior.

Recommended merge strategy:

- Only cherry-pick bug fixes and ergonomic improvements.
- Do not import upstream navigation, branding, or layout wholesale.

## Not Migratable

These files should not be copied from upstream into `flvxt2`, because they would break repository ownership, licensing, release assets, branding, commercial behavior, or custom product logic.

### Installation, Release, And Image Ownership

- `install.sh`
- `panel_install.sh`
- `docker-compose-v4.yml`
- `docker-compose-v6.yml`
- `.github/workflows/*`
- `go-backend/internal/http/handler/upgrade.go`
- `go-backend/internal/http/handler/system_upgrade.go`
- `vite-frontend/src/config/site.ts`
- `vite-frontend/src/components/version-footer.tsx`

Reason:

- Upstream points to `Sagit-chu/flvx`, `Sagit-chu/flux-panel`, and `ghcr.io/sagit-chu/...`.
- `flvxt2` must remain on `iKeilo/FLVXR2` and `ghcr.io/ikeilo/...`.
- Replacing these would recreate the earlier wrong-project installation problem.

Allowed action:

- Manually inspect upstream for installer bug fixes only.
- Keep all repo/image/version ownership values from `flvxt2`.

### Commercial License And Business Modules

Do not replace or backport upstream over these target-side files:

- `go-backend/internal/license/defaults.go`
- `go-backend/internal/license/domain.go`
- `go-backend/internal/middleware/license_check.go`
- `go-backend/internal/http/middleware/license_guard.go`
- `go-backend/internal/http/middleware/trial_guard.go`
- `go-backend/internal/http/handler/license_config.go`
- `go-backend/internal/http/handler/license_info.go`
- `go-backend/internal/http/handler/license_tier.go`
- `go-backend/internal/http/handler/license_commercial.go`
- `go-backend/internal/http/handler/license_transfer.go`
- `go-backend/internal/http/handler/billing.go`
- `go-backend/internal/http/handler/product.go`
- `go-backend/internal/http/handler/order.go`
- `go-backend/internal/http/handler/payment.go`
- `go-backend/internal/payment/*`
- `go-backend/internal/store/model/billing.go`
- `go-backend/internal/store/model/order.go`
- `go-backend/internal/store/model/product.go`
- `go-backend/internal/store/model/payment_config.go`
- `go-backend/internal/store/repo/repository_billing.go`
- `go-backend/internal/store/repo/repository_order.go`
- `go-backend/internal/store/repo/repository_payment_config.go`

Reason:

- These are `flvxt2` commercial product layers.
- Upstream does not have equivalent commercial semantics.

### Node Deployment, TLS, And Inbound Management

Do not replace these target-side files with upstream equivalents or remove them during merge:

- `go-backend/internal/http/handler/node_deploy.go`
- `go-backend/internal/store/repo/repository_node_deploy.go`
- `vite-frontend/src/pages/node/node-deploy-modal.tsx`
- `vite-frontend/src/pages/tls.tsx`
- `vite-frontend/src/api/types.ts` TLS/deploy sections

Reason:

- These implement S-UI-inspired TLS/Reality template and node deployment behavior.
- Upstream does not model this feature set.

### WG Path Product Surface

Do not replace these target-side files with upstream:

- `go-backend/internal/http/handler/wg_path.go`
- `go-backend/internal/http/handler/wg_path_test.go`
- `go-backend/internal/store/repo/repository_wg_path.go`
- `go-gost/x/socket/wireguard_handler.go`
- `go-gost/x/socket/wireguard_handler_stub.go`
- `vite-frontend/src/pages/tunnel/wg-path-manager.tsx`
- `vite-frontend/src/pages/forward.tsx` WG mode sections
- `vite-frontend/src/pages/tunnel.tsx` WG switch/manager sections

Reason:

- WG Path is a `flvxt2` feature and is admin-gated.
- Upstream does not contain the same model.

### Generated Or Binary Artifacts

Do not migrate these from either side:

- `docs/assets/*`
- `docs/*.html`
- `go-backend/cmd/paneld/paneld`
- `go-gost/flux_agent`
- `go-gost/gost-linux-amd64`
- `vite-frontend/dist/*`

Reason:

- These are generated build artifacts or binaries.
- They should be rebuilt by CI/release workflows.

## Recommended Execution Order

1. Import direct test files first.
2. Port low-risk backend utilities: `security_utils.go`, password helpers, config policy helpers.
3. Port nftables runtime as an isolated package behind a feature boundary.
4. Add Panel Peering compatibility adapter and tests for Sagit-chu baseline providers.
5. Port tunnel probe/best-exit fixes and expose them to WG diagnostics.
6. Port forward-mode fixes while preserving `wg_path` and admin-only restrictions.
7. Re-run backend tests, frontend build, and `go-gost` build.
8. Deploy to test server and verify:
   - normal tunnel creation
   - nftables mode
   - WG Path creation/edit/diagnosis
   - non-admin cannot create/use WG
   - login/logout still uses HttpOnly cookie
   - install scripts still resolve `iKeilo/FLVXR2`
   - imported Sagit-chu panel share can reserve/apply/release a GOST tunnel
   - imported Sagit-chu panel share does not expose WG/TLS deployment-only actions
