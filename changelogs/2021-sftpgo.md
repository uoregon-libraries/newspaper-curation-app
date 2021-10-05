## vX.Y.Z

SFTPGo integration and documentation

### Added

- [SFTPGo](https://github.com/drakkan/sftpgo) is now integrated with NCA for
  managing titles' uploads.

### Changed

- Users with the role "Title Manager" can now edit all aspects of a title,
  including SFTP data. Since we no longer store plaintext passwords, there's
  no reason to do the weird half-editing setup we had before where only admins
  could edit the SFTP stuff.

### Removed

- SFTP Password and SFTP directory fields are no longer stored in NCA, as
  neither of these fields had any way to tie to a backend SFTP daemon, and got
  out of sync too easily

### Migration

- Database migration, e.g.:
  - `goose -dir ./db/migrations/ mysql "<user>:<password>@tcp(<db host>:3306)/<database name>" up`
- Set up SFTPGo if desired. The docs cover
  [SFTPGo integration](https://uoregon-libraries.github.io/newspaper-curation-app/setup/sftpgo-integration/),
  including how to *not* integrate with SFTPGo.
  - Note that if you don't integrate, but had been relying on the SFTP fields,
    you will lose this functionality. Due to maintenance difficulties and
    complexity in trying to wrangle conditional use of this data, NCA will no
    longer manage those fields or even display them.
- If you switch from a traditional sftp daemon to sftpgo, there will be a
  service disruption publishers need to be made aware of.
