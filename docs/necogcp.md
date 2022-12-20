necogcp
=======

`necogcp` is a command-line tool for GCP provisioning.

Synopsis
--------

### GCE instance management on developer's project

* `necogcp create-image`

    Build `vmx-enabled` image.
    If `vmx-enabled` image already exists, it is re-created.

* `necogcp create-instance`

    Launch `host-vm` instance using `vmx-enabled` image.
    If `host-vm` instance already exists, it is re-created.

* `necogcp create-runner`

    Launch runner instance which runs self-hosted-runner with `vmx-enabled` image.
    If runner instance already exists in the project, new runner is not created
    You must have ServiceAccount which has permission `secretmanager.versions.access` in your project.

* `necogcp delete-image`

    Delete `vmx-enabled` image.

* `necogcp delete-instance`

    Delete `host-vm` instance.

* `necogcp setup-instance`

    Setup `host-vm` or `vmx-enabled` instance. It can run on only them.

* `necogcp create-snapshot`

    Create `home` volume snapshot. This subcommand is mainly for backup purpose.

* `necogcp restore-snapshot`

    Restore `home` volume from the latest snapshot in the zone specified by the flag `--dest-zone`. You have to delete the `home` persistent disk manually before restoration.

### GCE instance management on neco-test project

* `necogcp neco-test create-image`

    Build `vmx-enabled` image on neco-test.
    If `vmx-enabled` image already exists, it is re-created.

* `necogcp neco-test delete-image`

    Delete `vmx-enabled` image on neco-test.

* `necogcp neco-test extend INSTANCE_NAME`

    Extend 2 hours given instance on the neco-test project to prevent deleted by GAE app.

The subcommands `necogcp neco-test create-instance | delete-instance | list-instance` are used for `auto-dctest` feature. See [auto-dctest.md](auto-dctest.md)

### Miscellaneous

* `necogcp completion`

    Dump bash completion rules for `necogcp` command.

Flags
-----

| Flag          | Default value        | Description                                                                      |
| ------------- | -------------------- | -------------------------------------------------------------------------------- |
| `--config`    | `$HOME/.necogcp.yml` | [Viper configuration file](https://github.com/spf13/viper#reading-config-files). |
| `--dest-zone` | `an empty string`    | zone to restore `home` volume                                                    |

Configuration file
------------------

See [config.md](config.md)
