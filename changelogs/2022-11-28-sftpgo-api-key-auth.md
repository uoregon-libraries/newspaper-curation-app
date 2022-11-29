### Added
- Bash scripts added to `sftpgo/` to manage SFTPGo API keys

### Changed
- SFTPGo integration now uses an admin API key for auth rather than short term
  tokens

### Migration
- If using SFTPGo integration, `settings` now requires `SFTPGO_ADMIN_API_KEY`
  and drops `SFTPGO_ADMIN_LOGIN` and `SFTPGO_ADMIN_PASSWORD`. Copy the relevant
  section from `settings-example` and run `sftpgo/get_admin_api_key.sh` to issue
  an admin API key and automatically store it in `settings`.

