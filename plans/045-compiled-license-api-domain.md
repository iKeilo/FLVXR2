# 045-compiled-license-api-domain

- [x] Update FLVX backend source so the default license API endpoint is compiled as `https://sq.sbplay.eu.org`.
- [x] Remove `LICENSE_SERVER_URL` from panel deployment environment so the container does not depend on it.
- [x] Rebuild and redeploy the FLVX backend, then verify license checks still pass through the compiled endpoint.
