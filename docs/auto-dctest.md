Automatic Neco Environment Construction with GCP (auto-dctest)
==============================================================

Overview
--------

This document describes the system which bootstraps [Neco](https://github.com/cybozu-go/neco) environment automatically on Google Cloud Platform(GCP).

### Features

1. Automatic Neco environment construction at the fixed time every buisiness day
2. Automatic deletion of the environment
3. Manual management of the environment with `necogcp` command
4. Slack notification when the environment is created/deleted
5. Multi-team support

### Architecture

![diagram](./images/auto-dctest.png)

This system consists of the following two types of components:

- `auto-dctest`: create/delete VM instances
  - Cloud Scheduler(to create VM): triggered every 30 minutes between 9:00AM and 8:00PM
  - Cloud Scheduler(to delete VM): triggered at 8:00PM
  - Cloud Scheduler(to force-delete VM): triggered at 11:00PM
  - Cloud Pub/Sub: messaging queue to accept messages from Cloud Scheduler
  - Cloud Function: workload to create/delete VM instances
    - This will create the specified number of instances for each team.  
    - Instances are named with the format `<team_name>-<index>`.  
      For example, `sample-0` and `sample-1` are created if team_name = sample and
      instance num = 2.  
      If `sample-0` already exists before creating, this function does nothing for it.
    - Neco/[neco-apps](https://github.com/cybozu-go/neco-apps) are started using (startup-script)[https://cloud.google.com/compute/docs/startupscript]
- `slack-notifier`: notify messages via Slack
  - Cloud Logging Sink: filters the log and push events to Cloud Pub/Sub
  - Cloud Pub/Sub: messaging queue to accept messages from Cloud Logging Sink
  - Cloud Function: workload to notify messages via Slack

`slack-notifier` notifies messages based on the texts in the logs pushed by `auto-dctest`.
`auto-dctest` and `slack-notifier` can be used independently, which means that `auto-dctest` can be used
without `slack-notifier` and `slack-notifier` can be used without `auto-dctest`.

Usage
-----

Note that the word "instance" means the VM instance on Google Compute Engine(GCE).

### Quota

These quotas are relevant to this system.
To avoid paying huge and unintended cost, limiting these quotas are strongly recommended.

The functions are assumed to be deployed in the asia-northeast1 region.

#### Cloud Functions API

- Incoming socket traffic for asia-northeast1 (deprecated) per 100 seconds
  - This depends on how many teams you want manage.
  - One traffic from Pub/Sub is expected to be within 100kB.
  - If you manage 10 teams, 100kB * 10 = 1MB is minimal.
- CPU allocation in function invocations for asia-northeast1 (deprecated) per 100 seconds
  - `auto-dctest` and `slack-notifier` run with 200MHz(=0.2GHz) CPU.
  - This depends on how many teams you want manage.
  - Both `auto-dctest` and `slack-notifier` are expected to finish within 30s.
  - 0.2GHz * 30s * (N of teams) or more is required to be set.
  - If you manage 10 teams, 60 GHz * s is minimal.

#### Cloud Pub/Sub

- Regional push subscriber throughput, kB per minute
  - One log is expected to be within 5kB.
  - 5 ~ 10 logs are filtered by a single run of `auto-dctest`.
  - This depends on how many teams you want manage.
  - 5kB * 10 logs * (N of teams) or more is required to be set.
  - If you manage 10 teams, 500kB is minimal.

#### Cloud Scheduler

- Requests per minute
  - This depends on how many teams you want manage.
  - If you manage 10 teams, 10 is minimal.

#### Secret Manager API

- Access requests per minute
  - This depends on how many instances you want to create/delete.
    The total number of instances is calculated by the sum of the instances of each team.
  - At most, 2 logs by the `auto-dctest` Cloud Function invokes this API in a minute.
  - 2 logs * total instances or more is required to be set.
  - If you manage 10 teams and each team has 3 instances, 60 is minimal.

### Deploy `auto-dctest` function

Deploy `auto-dctest` function and schedulers for deletion:
```
export GCP_PROJECT=<project>
make -f Makefile.dctest init`
```

Create a scheduler for creation per each team:
```
export GCP_PROJECT=<project>
make -f Makefile.dctest add-team TEAM_NAME=<team_name> INSTANCE_NUM=<num>
make -f Makefile.dctest list-teams
```

When you want to destroy:
```
export GCP_PROJECT=<project>
make -f Makefile.dctest list-teams
make -f Makefile.dctest delete-team TEAM_NAME=<team_name>
make -f Makefile.dctest clean
```

If you want to use Contour or ExternalDNS by manually creating `HTTPProxy` or `DNSEndpoint`,
you should deploy the JSON key file of the Service Account which has an editor access to CloudDNS of neco-dev.
The file is deployed onto Secret Manager with the name `cloud-dns-admin-account`.

Currently, `neco-apps@neco-dev.iam.gserviceaccount.com` exists in the neco-dev project, and it has the editor access to CloudDNS.

### Deploy `slack-notifier` function

Deploy `slack-notifier` function and logging sink:
```
export GCP_PROJECT=<project>
make -f Makefile.slack init`
```

When you want to destroy:
```
export GCP_PROJECT=<project>
make -f Makefile.slack clean`
```

### Slack notifications

`auto-dctest` can notify the following events via Slack:

1. Starting instance creation
2. Finished `cybozu-go/neco` DC test bootstrap
3. Finished `cybozu-go/neco-apps` DC test bootstrap
4. Deleting the instance

To enable Slack notification, you need to prepare a YAML setting file:

```yaml
teams:
  team1: https://<your>/<slack>/<webhook>/<url>
severity: # See https://api.slack.com/reference/messaging/attachments#fields
  - color: good
    regex: ^INFO
  - color: warning
    regex: ^WARN
  - color: danger
    regex: ^ERROR
rules:
  - name: team1-rule
    regex: team1-[0-9]+
    excludeRegex: 'team1-0'
    targetTeams:
      - team1
```

With the above setting, the notifications for `team1`'s instance with postfix `-[0-9]+` except for `team1-0` will be sent to `team1`'s Slack webhook.

This YAML file must be uploaded to Secret Manager with the specific name `slack-notifier-config`.

You should save the YAML file as `slack-notifier-config.yaml` and run the following commands.
```
export GCP_PROJECT=<project>
make create-slack-notifier-config -f Makefile.slack
```

### Manual management with `necogcp` command

Neco environment can be created with `necogcp neco-test` commands.

#### `create-instance`

| Flag (short)      | Default Value                          | Description                   |
| :---------------- | :------------------------------------- | :---------------------------- |
| project-id (p)    | -                                      | Project ID for GCP (required) |
| zone (z)          | asia-northeast1-c                      | Zone name for GCP             |
| machine-type (t)  | n1-standard-64                         | VM Machine type               |
| local-ssd (s)     | 4                                      | Number of local SSDs(*)       |
| instance-name (n) | -                                      | Instance name (required)      |
| neco-branch       | Branch of `cybozu-go/neco` to run      | release                       |
| neco-apps-branch  | Branch of `cybozu-go/neco-apps` to run | release                       |

(*) There are constraints around how many local SSDs you can attach based on each machine type.
See the [GCE documentation](https://cloud.google.com/compute/docs/disks#local_ssd_machine_type_restrictions).

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

### Administration

#### Holiday list

The scheduler for `auto-dctest` skips weekend (Saturday and Sunday) and holidays.

The holiday list is hard-coded in [`config_autodctest.go`](../config_autodctest.go),
so an administrator should modify it periodically.

#### Team management

To add a team, an administrator should run as follows
```
export GCP_PROJECT=<project>
make -f Makefile.dctest add-team TEAM_NAME=<team name> INSTANCE_NUM=<max instance number>
# create slack-notifier-config.yaml
make update-slack-notifier-config -f Makefile.slack
```

### Development

[`cmd/dev`](../cmd/dev) includes commands, which are equivalent to the Cloud Functions executed by `auto-dctest`.
These are useful if you want to debug the Cloud Function without deploying.

### CI

To avoid the cloud functions(i.e. `slack-notifier` and `auto-dctest`) being outdated, they are recreated in CI nightly.
