# Slack CLI

A cli for interacting with slack from a terminal.

## Requirements

- OSX only (possible to support linux in the future)
- Slack desktop app installed
  - Credentials are borrowed from the slack app after it has authenticated

## Installation

The cli can be installed by downloading the appropriate binary from the releases tab or by using any of the supported installation methods.

### Go Install

```bash
go install github.com/graytonio/slack-cli@latest
```

## Setup

On first run the cli will create a configuration file at `~/.config/slackcli.yaml` with a blank configuration file. It will then output an error that the credentials are not configured. Open the configuration file and fill in the workspace key with the name of the workspace you want the cli to connect to.

For example if your slack workspace is `my-workspace.slack.com` then you would fill in the configuration file to look like:

```yaml
workspace: "my-workspace"
```

_NOTE_ Before running the cli again make sure to fully quit the slack desktop application. The credentials are extracted from the local storage of the slack app which is locked if it is running and stores it in the configuration file for future use. Once this has happened the app can be open without interfering with the functionality of the cli.

## Usage

### Send Message

Sending a message to a user or to a channel by using the id of the conversation or a saved alias.

**Example**

```bash
# Basic Usage
slack-cli send C12341234 "Hello this is my message"

# Use stdin as message
echo "Hello from my terminal" | slack-cli send C12341234 -

# Use user alias
slack-cli send @username "Hello username"

# Use channel alias
slack-cli send "#my-channel-name" "Hello team"
```

### Save Alias

Since the slack api does not support looking up a user or channel by it's name the cli supports saving a user or channel id as an alias for later use. These aliases can later be used in other commands like the send command

**Example**

```bash
# Save User
slack-cli save user my-user U12341234

# Save Channel
slack-cli save channel my-team-chat C12341234
```

### Sort Channels

You can organize your channels into sections automatically using regex to identify which channels should go into which section.

**Example**

```bash
# Sort all channels that match regex 'incident-' into the section called "Incident Channels"
slack-cli sort "Incident Channels" "incident-"

# Sort the section "Incident Channels" using the regex found in the config file
slack-cli sort "Incident Channels"

# Sort all sections in the config file
slack-cli sort
```

## Configuration

The configuration file is stored at `${HOME}/.config/slackcli.yaml` and is read on each cli execution.

| Key                    | Documentation                                                                     | Default |
|------------------------|-----------------------------------------------------------------------------------|---------|
| credentials            | Stores the extracted credentials for authenticating with slack                    | null    |
| credentials.cookie     | Extracted d cookie                                                                | ""      |
| credentials.token      | User authentication token                                                         | ""      |
| workspace              | Workspace identifier to send api calls to                                         | ""      |
| smart_sections         | Array of smart section configurations                                             | []      |
| smart_sections.re      | Regex to run against channel name to know if it should be matched to this section | ""      |
| smart_sections.section | Section to put matching channels in. Does not need to already exist               | ""      |
| users_cache            | A dictionary to match a given user alias ("@alias") to a known channel id         | null    |
| channel_cache          | A dictionary to match a given channel alias ("#alias") to a known channel id      | null    |

### Example Configuration

```yaml
channel_cache:
    team: C0000000000
credentials:
    cookie: "<cookie>"
    token: "<token>"
smart_sections:
    - re: incident-
      section: PDE
    - re: \d\d\d\d-\d\d-\d\d-
      section: Incidents
    - re: my-team
      section: My Team
users_cache:
    me: D00000000
workspace: my-workspace
```
