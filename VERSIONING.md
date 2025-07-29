# Versioning Scheme for the SUSE Telemetry Gateway services

This repository implements a Go style semantic versioning model, meaning
that there is `v` prefix to the version, with major, minor and patch level
fields along with optional prerelease and build fields, as follows:

```
v<MAJOR>.<MINOR>.<PATCH>[-<PRERELEASE>][+<BUILD>]
```

## Version Location

The version is stored in the [app/VERSION](app/VERSION) file, and pulled
into the code at runtime using the Go embed module, and accessible via
the `GetVersion()` call in the `app` module.

## Version Management

A helper tool, `versionbump`, is implemented in [bin/versionbump](bin/versionbump),
and is available to assist in managing the [app/VERSION](app/VERSION) file.

This tool is inspired by Python's [bumpversion](https://pypi.org/project/bump2version/)
but implemented in Go.

The `versionbump` tool takes as argument an action to be performed,
along with options that can be used to set the `<PRERELEASE>` and
`<BUILD>` fields:
* `update`
  - update, or clear, the `<PRERELEASE>` and `<BUILD>` fields.
* `patch`
  - increment the `<PATCH>` field.
  - update, or clear, the `<PRERELEASE>` and `<BUILD>` fields.
* `minor`
  - increment the `<MINOR>` field
  - reset the `<PATCH>` field to `0`.
  - update, or clear, the `<PRERELEASE>` and `<BUILD>` fields.
* `major`
  - increment the `<MAJOR>` field
  - reset the `<MINOR>` field to `0`.
  - reset the `<PATCH>` field to `0`.
  - update, or clear, the `<PRERELEASE>` and `<BUILD>` fields.

The `versionbump` tools supports a dryrun option that can be used
to see how a version would be updated without making any changes.

# Release Versioning Scheme

For the SUSE Telemetry Gateway services a release should be built
with a version that consists of the base major, minor and patch
fields along with the build field set to the UTC date on which the
release is cut.

The release should be tagged with version that consists of the base
major, minor and patch fields.

Once the release is cut and tagged, the version should be bumped
appropriately, usually at the patch level, with a `dev` prerelease
value.

This versioning scheme will ensure that any subsequent developer
builds will have a newer version than any released version, with
the prerelease value indicating that it is a developer build.

## Makefile support for release version management

A Makefile rule `release` is provided that will automate the above
release versioning scheme, though by default it will not make any
changes because the `VERSIONBUMP_NOCHANGE` parameter is set to true.

Additionally the `VERSIONBUMP_BUMP_MODE` parameter can be used to
control which field of the version field gets bumped.

With the `VERSIONBUMP_NOCHANGE` parameter set to true, running the
`release` rule will perform the appropriate `versionbump` operations,
interspersed with printing the `git` commands that would be used to
commit those version updates, tag the release and push these changes
upstream to the GitHub repository.

The `release` rule checks that the active branch is the appropriate
one as specified by the `VERSIONBUMP_BRANCH`.