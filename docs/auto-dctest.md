Automatic Neco Environment Construction with GCP (auto-dctest)
==============================================================

Overview
--------

diagram

`auto-dctest` provides automatic Neco environment construction with GCP services.

### Features

1. Automatic Neco environment construction at the fixed time every day (at 8:00 AM)
2. Automatic deletion of the environment (at 8:00 PM or 12:00 AM)
3. Manual management of the environment with `necogcp` command
4. Slack notification when the environment is created/deleted
5. Multi-team support

Usage
-----

### Deploy `auto-dctest` Function

Create `auto-dctest` function and schedulers for deletion:
```
export GCP_PROJECT=<project>
export ZONE=<zone> # If needed
make -f Makefile.dctest init`
```

Create a scheduler for creation per each team:
```
export TEAM_NAME=<team>
export INSTANCE_NUM=<num>
make -f Makefile.dctest add-team
```

When you want to destroy:
```
export GCP_PROJECT=<project>
export ZONE=<zone> # If needed
export TEAM_NAME=<team>
make -f Makefile.dctest delete-team
make -f Makefile.dctest clean
```

### Deploy `slack-notifier` Function

Create `slack-notifier` function and logging sink:
```
export GCP_PROJECT=<project>
export REGION=<region> # If needed
make init
```

When you want to destroy:
```
export GCP_PROJECT=<project>
export REGION=<region> # If needed
make clean
```

### Manual management with `necogcp` command

Neco environment can be created with `necogcp neco-test` commands.

#### `create-instance`

| Flag (short)      | Default Value                          | Description                   |
| :---------------- | :------------------------------------- | :---------------------------- |
| project-id (p)    | -                                      | Project ID for GCP (required) |
| zone (z)          | asia-northeast1-c                      | Zone name for GCP             |
| machine-type (t)  | n1-standard-32                         | VM Machine type               |
| instance-name (n) | -                                      | Instance name (required)      |
| neco-branch       | Branch of `cybozu-go/neco` to run      | release                       |
| neco-apps-branch  | Branch of `cybozu-go/neco-apps` to run | release                       |

#### `delete-instance`

| Flag (short)      | Default Value     | Description                   |
| :---------------- | :---------------- | :---------------------------- |
| project-id (p)    | -                 | Project ID for GCP (required) |
| zone (z)          | asia-northeast1-c | Zone name for GCP             |
| instance-name (n) | -                 | instance name (required)      |

#### `list-instances`

| Flag (short)   | Default Value     | Description                   |
| :------------- | :---------------- | :---------------------------- |
| project-id (p) | -                 | Project ID for GCP (required) |
| zone (z)       | asia-northeast1-c | Zone name for GCP             |
| filter (f)     | -                 | Filter string                 |
