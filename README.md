
# ssh-sync: Seamless SSH Key Management

ssh-sync is a powerful CLI tool designed to simplify the way you manage and synchronize your SSH keys and configurations across multiple machines. With ssh-sync, gone are the days of manually copying SSH keys or adjusting configurations when switching devices. Whether you're moving between workstations or setting up a new machine, ssh-sync ensures your SSH environment is up and running effortlessly.

[![release](https://github.com/therealpaulgg/ssh-sync/actions/workflows/release.yml/badge.svg)](https://github.com/therealpaulgg/ssh-sync/actions/workflows/release.yml)

## Quick Start

### Installation

ssh-sync is available on Windows, macOS, and Linux. Choose the installation method that best suits your operating system:

#### Windows

Install ssh-sync using Winget:

```shell
winget install ssh-sync
```

#### macOS

ssh-sync can be installed using Homebrew:

```shell
brew tap therealpaulgg/ssh-sync
brew install ssh-sync
```

#### Linux

For Linux users, download the appropriate package from our [GitHub Releases](https://github.com/therealpaulgg/ssh-sync/releases) page:

- For Debian-based distributions (e.g., Ubuntu):

```shell
wget <link-to-.deb-file>
sudo dpkg -i ssh-sync_0.3.8_amd64.deb
```

- For RPM-based distributions (e.g., Fedora, CentOS):

```shell
wget <link-to-.rpm-file>
sudo rpm -i ssh-sync-v0.3.8-1.x86_64.rpm
```

## Getting Started with SSH-Sync

SSH-Sync makes managing and syncing your SSH keys across multiple machines effortless. Here's how to get started:

### Setup

First, you'll need to set up SSH-Sync on your machine. Run the following command:

```shell
ssh-sync setup
```

During setup, you'll be prompted to choose between using your own server or the sshsync.io hosted server. Next, you'll specify whether you have an existing account. If you do not, you'll be guided through creating an account, naming your machine, and generating a keypair for it. If you have an existing account, you'll be given a challenge phrase, which you must enter on another one of your machines using the `challenge-response` command. This process securely adds your new machine to your SSH-Sync account.

### Uploading Keys

To upload your SSH keys and configuration to the server, run:

```shell
ssh-sync upload
```

This command securely transmits your SSH keys and configuration to the chosen server, making them accessible from your other machines.

### Downloading Keys

To download your SSH keys to a new or existing machine, ensuring it's set up for remote access, use:

```shell
ssh-sync download
```

This command fetches your SSH keys from the server, setting up your SSH environment on the machine.

### Challenge Response

If setting up a new machine with an existing account, use:

```shell
ssh-sync challenge-response
```

Enter the challenge phrase received during the setup of another machine. This verifies your new machine and securely transfers the necessary keys.

### Managing Machines

To list all machines configured with your SSH-Sync account, run:

```shell
ssh-sync list-machines
```

If you need to remove a machine from your SSH-Sync account, use:

```shell
ssh-sync remove-machine
```

Specify the machine you wish to remove following the command.

### Reset

To remove the current machine from your account and clear all SSH-Sync data:

```shell
ssh-sync reset
```

This command is useful if you're decommissioning a machine or wish to start fresh.

By following these steps, you can seamlessly sync and manage your SSH keys across all your machines with SSH-Sync.

## How ssh-sync Works

ssh-sync leverages a client-server model to store and synchronize your SSH keys securely. The diagram below outlines the ssh-sync architecture and its workflow:

![ssh-sync Architecture](https://raw.githubusercontent.com/therealpaulgg/ssh-sync/main/docs/diagrams.svg)

For a deep dive into the technicalities of ssh-sync, including its security model, data storage, and key synchronization process, check out our [Wiki](https://github.com/therealpaulgg/ssh-sync/wiki).

## Why Choose ssh-sync?

- **Simplify SSH Key Management:** Easily sync your SSH keys and configurations across all your devices.
- **Enhanced Security:** ssh-sync uses advanced cryptographic techniques to ensure your SSH keys are securely transmitted and stored.
- **Effortless Setup:** With support for Windows, macOS, and Linux, setting up ssh-sync is straightforward, regardless of your operating system.

## Contributing

ssh-sync is an open-source project, and contributions are welcome! If you're interested in contributing, please check out our [contribution guidelines](https://github.com/therealpaulgg/ssh-sync/blob/main/CONTRIBUTING.md).

## License

ssh-sync is released under the [MIT License](https://github.com/therealpaulgg/ssh-sync/blob/main/LICENSE.txt).

