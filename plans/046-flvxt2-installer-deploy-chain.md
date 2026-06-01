# FLVX flvxt2 Installer Deploy Chain

- [x] Point panel and node installers at `iKeilo/flvxt2`.
- [x] Point Docker Compose defaults and cleanup logic at `ghcr.io/ikeilo`.
- [x] Replace legacy license stats domain with `sq.sbplay.eu.org`.
- [x] Add a raw-main compose fallback for first deployments before a release exists.
- [x] Enable `main` pushes to build and publish `latest` images.
- [x] Run available static checks and confirm no authorization server files are included.

Note: local `bash -n` validation could not run because this Windows environment has no Bash/WSL distro installed.
