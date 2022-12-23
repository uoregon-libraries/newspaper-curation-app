---
title: Add Configuration Settings
weight: 20
description: Adding a new configuration setting
---

Occasionally we need a new setting to be created so that users have a bit more
control over the inner workings of NCA. This details the process of adding
settings:

- Open up [`src/config/config.go`][1] and add a value to the Config struct.
- Choose the data type. In most cases a primitive is fine: string, int,
  float64, etc.
- Decide if the value should be pulled directly from the settings file or if
  you need to massage data manually. The former is usually the best option, but
  not always possible.
- If the value is pulled directly from settings, set up the struct tags:
  - At a minimum you must define which settings value will populate the Go
    config structure; e.g., the `config.Ghostscript` value specifies
    `setting:"GHOSTSCRIPT"` in the struct tag, telling us the `settings` file's
    "GHOSTSCRIPT" value is to be used.
  - If you want validation, use a "type" struct tag, e.g., the "Webroot"
    setting uses `type:"url"` to specify that the value *must* be a valid URL.
- If the value is not directly pulled from settings, modify `Parse()` to read
  the raw setting and set the config field accordingly.
- Open `settings-example` and add the setting with some documentation
  (bash-style comments) explaning what it does and how it should be used. When
  you can, make sure the default "just works" with a standard docker setup.

[1]: <https://github.com/uoregon-libraries/newspaper-curation-app/blob/main/src/config/config.go>
