Closes #

(Describe changes as necessary)

## Normal contributors

I have done all of the following:

- [ ] Fixes and new features have unit tests where applicable
- [ ] A new changelog has been created in `changelogs/` (based on
  [`changelogs/template.md`][1])
- [ ] Documentation has been updated as necessary (`hugo/content/`)

[1]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/changelogs/template.md>

## Release deployer

I have done all of the following:

- [ ] Put the contents of `changelogs/*` into `CHANGELOG.md`, rewording as
  necessary
- [ ] Delete `changelogs/*` (not the template of course)
- [ ] Set an appropriate version number in the changelog per semantic
  versioning specs
- [ ] Compiled the hugo documentation and verified very carefully that it is
  correctly generated
- [ ] Tested the code carefully, thoroughly, meticulously, and lovingly. It is
  production-ready.

Once this merges, *I swear on all I hold dear* not to forget any of the
post-deploy steps. I will set up a reminder in Outlook, gmail, via some smart
device, etc.

- [ ] Create and push a tag
- [ ] Create a github release from aforementioned tag, describing the changes
  briefly and linking to the full changelog.
