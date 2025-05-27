
# ssh-sync: Seamless SSH Key Management

ssh-sync is a powerful CLI tool designed to simplify the way you manage and synchronize your SSH keys and configurations across multiple machines. With ssh-sync, gone are the days of manually copying SSH keys or adjusting configurations when switching devices. Whether you're moving between workstations or setting up a new machine, ssh-sync ensures your SSH environment is up and running effortlessly.

[![release](https://github.com/therealpaulgg/ssh-sync/actions/workflows/release.yml/badge.svg)](https://github.com/therealpaulgg/ssh-sync/actions/workflows/release.yml)

## Quick Start

### Installation

ssh-sync is available on Windows, macOS, and Linux. Choose the installation method that best suits your operating system:

#### Windows

Install ssh-sync using Winget:

```shell
winget install therealpaulgg.ssh-sync
```

#### macOS

ssh-sync can be installed using Homebrew:

```shell
brew tap therealpaulgg/ssh-sync
brew install ssh-sync
```

#### Linux

For Linux users, you can install ssh-sync through our official package repositories or by downloading packages directly from our [GitHub Releases](https://github.com/therealpaulgg/ssh-sync/releases) page:

##### Using the Official Repository

###### Debian/Ubuntu and derivatives:

```shell
# Import the GPG key
curl -fsSL https://repo.sshsync.io/ssh-sync-repo.asc | sudo gpg --dearmor -o /usr/share/keyrings/ssh-sync-archive-keyring.gpg

# Add the repository
echo "deb [signed-by=/usr/share/keyrings/ssh-sync-archive-keyring.gpg] https://repo.sshsync.io/debian $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/ssh-sync.list

# Update package lists
sudo apt update

# Install ssh-sync
sudo apt install ssh-sync
```

###### Fedora/RHEL/CentOS and derivatives:

```shell
# Import the GPG key
sudo rpm --import https://repo.sshsync.io/ssh-sync-repo.asc

# Add the repository
cat <<EOF | sudo tee /etc/yum.repos.d/ssh-sync.repo
[ssh-sync]
name=SSH-Sync Repository
baseurl=https://repo.sshsync.io/rpm
enabled=1
gpgcheck=1
gpgkey=https://repo.sshsync.io/ssh-sync-repo.asc
EOF

# Install ssh-sync
sudo dnf install ssh-sync
```

##### Manual Installation

If you prefer to download and install the package manually:

- For Debian-based distributions (e.g., Ubuntu):

```shell
wget https://github.com/therealpaulgg/ssh-sync/releases/latest/download/ssh-sync_VERSION_ARCH.deb
sudo dpkg -i ssh-sync_VERSION_ARCH.deb
```

- For RPM-based distributions (e.g., Fedora, CentOS):

```shell
wget https://github.com/therealpaulgg/ssh-sync/releases/latest/download/ssh-sync-VERSION-ARCH.rpm
sudo rpm -i ssh-sync-VERSION-ARCH.rpm
```

- For Alpine Linux:

```shell
wget https://github.com/therealpaulgg/ssh-sync/releases/latest/download/ssh-sync_VERSION_ARCH.apk
sudo apk add --allow-untrusted ssh-sync_VERSION_ARCH.apk
```

- For Arch-based distributions (e.g., Arch Linux, Manjaro):

```shell
wget https://github.com/therealpaulgg/ssh-sync/releases/latest/download/ssh-sync-VERSION-ARCH.tar.zst
sudo pacman -U ssh-sync-VERSION-ARCH.tar.zst
```

We provide packages for multiple architectures including:
- x86_64 (amd64)
- ARM64 (aarch64)
- ARMv7

Choose the appropriate package for your system architecture.

## Getting Started with SSH-Sync

SSH-Sync makes managing and syncing your SSH keys across multiple machines effortless. Here's how to get started:

### Setup

First, you'll need to set up SSH-Sync on your machine. Run the following command:

```shell
ssh-sync setup
```

During setup, you'll be prompted to choose between using your own server or the sshsync.io hosted server. Next, you'll specify whether you have an existing account. If you do not, you'll be guided through creating an account, naming your machine, and generating a keypair for it. If you have an existing account, you'll be given a challenge phrase, which you must enter on another one of your machines using the `challenge-response` command. This process securely adds your new machine to your SSH-Sync account.

### Uploading Keys

**NOTE**: Uploading keys will overwrite your existing keys on the server. Later versions of ssh-sync may include a conflict resolution for upload, but it is currently not implemented.

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

#### Conflict Resolution

In case there is a difference between a local file and one on your server, ssh-sync will let you know and you can opt to overwrite, skip, or save a file with a `.duplicate` extension for you to review manually.

```shell
diff detected for my_key.
1. Overwrite
2. Skip
3. Save new file (as .duplicate extension for manual resolution)
Please choose an option (will skip by default):
```

#### Safe Mode

If you simply want to download your keys to a temporary directory, and not interfere with your primary .ssh directory, you may also do this by using the `--safe-mode` (or `-s`) flag.

```shell
ssh-sync download --safe-mode
```

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
ssh-sync remove-machine <machine-name>
```

Specify the machine you wish to remove following the command.

You may optionally provide the machine name on the command line so you don't have to type it in when running the command.

### Reset

To remove the current machine from your account and clear all SSH-Sync data:

```shell
ssh-sync reset
```

This command is useful if you're decommissioning a machine or wish to start fresh.

By following these steps, you can seamlessly sync and manage your SSH keys across all your machines with SSH-Sync.

## Self-Hosting ssh-sync-server

In general, for self-hosting, we recommend a setup where ssh-sync-server is behind a reverse proxy (i.e Nginx), and SSL is handled via LetsEncrypt.

### Docker

Docker is the easiest way to run the server. Here is a simple `docker-compose` file you can use:

```yaml
version: '3.3'
services:
    ssh-sync-server:
        restart: always
        environment:
          - PORT=<your_port_here>
          - NO_DOTENV=1
          - DATABASE_USERNAME=sshsync
          - DATABASE_PASSWORD=${POSTGRES_PASSWORD}
          - DATABASE_HOST=ssh-sync-db:5432
        logging:
          driver: json-file
          options:
            max-size: 10m
        ports:
          - '<host_port>:<container_port>'
        image: therealpaulgg/ssh-sync-server:latest
        container_name: ssh-sync-server
    ssh-sync-db:
        image: therealpaulgg/ssh-sync-db:latest
        container_name: ssh-sync-db
        volumes:
          - /path/to/db-volume:/var/lib/postgresql/data
        environment:
          - POSTGRES_USER=sshsync
          - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
          - POSTGRES_DB=sshsync
        restart: always
```

### Nginx

Example Nginx config (must support websockets)

```nginx
server {
    listen [::]:443 ssl ipv6only=on; # managed by Certbot
    listen 443 ssl; # managed by Certbot
    ssl_certificate /etc/letsencrypt/live/server.sshsync.io/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/server.sshsync.io/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem; # managed by Certbot
    server_name server.sshsync.io;
    location / {
            proxy_pass http://127.0.0.1:<ssh-sync-server-port>;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "Upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Forwarded-For $remote_addr;
            proxy_set_header X-Real-IP $remote_addr;
    }


}
server {
    if ($host = server.sshsync.io) {
        return 301 https://$host$request_uri;
    } # managed by Certbot


    listen 80;
    listen [::]:80;
    server_name server.sshsync.io;
    return 404; # managed by Certbot


}
```

If you don't want to use docker, other methods of running are not supported at this time, but the source repos are linked below so you can configure your own server as you wish.

[ssh-sync-server Github](https://github.com/therealpaulgg/ssh-sync-server) 
[ssh-sync-db](https://github.com/therealpaulgg/ssh-sync-db)

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

