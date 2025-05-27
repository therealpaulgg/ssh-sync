# Repository Server Setup Guide

This document outlines the steps required to set up a server to host Debian and RPM package repositories for ssh-sync.

## Server Requirements

- A Linux server with SSH access
- Minimum 1GB RAM, 10GB storage
- Nginx or Apache web server
- Basic understanding of Linux package repositories
- GPG key for package signing

## Required GitHub Secrets

The following secrets need to be added to your GitHub repository settings for the workflow to successfully upload and sign packages:

- `REPO_SERVER_HOST`: The hostname or IP address of your package repository server
- `REPO_SERVER_USER`: The username for SSH access to the server
- `REPO_SERVER_SSH_KEY`: The private SSH key that allows the GitHub Action to authenticate with your server
- `REPO_SERVER_PORT`: The SSH port (usually 22)
- `REPO_SERVER_PATH`: The base path where package repositories will be stored on the server
- `GPG_KEY_ID`: The GPG key ID used for signing packages and repositories
- `REPO_SERVER_GPG_PASSPHRASE`: The passphrase for your GPG key

## Initial Server Setup

### 1. Directory Structure

Create the necessary directories for the repositories:

```bash
# Login to your server
ssh user@your-server

# Create main repository directory
sudo mkdir -p /var/www/repo

# Create specific directories for each repository type
sudo mkdir -p /var/www/repo/debian
sudo mkdir -p /var/www/repo/rpm

# Set appropriate permissions
sudo chown -R $USER:$USER /var/www/repo
sudo chmod -R 755 /var/www/repo
```

### 2. Install Required Software

#### For Debian Repository Management:

```bash
sudo apt-get update
sudo apt-get install -y dpkg-dev apt-utils gnupg2
```

#### For RPM Repository Management:

```bash
# On Debian/Ubuntu
sudo apt-get install -y createrepo-c gnupg2 rpm

# On Fedora/RHEL/CentOS
sudo dnf install -y createrepo_c gnupg2
```

### 3. GPG Key Setup

Generate a GPG key for signing packages:

```bash
# Generate a new GPG key
gpg --full-generate-key
```

Follow the prompts to create a GPG key. For repository signing, we recommend:
- Key type: RSA and RSA
- Key size: 4096 bits
- Key validity: 2 years or more
- Use a strong passphrase

After creating the key, list your keys and note the key ID:

```bash
gpg --list-secret-keys --keyid-format LONG
```

You'll see output like:
```
sec   rsa4096/3AA5C34371567BD2 2022-03-10 [SC]
      42B317FD4BA89E7A2BEBB8D73AA5C34371567BD2
uid                 [ultimate] Your Name <your.email@example.com>
```

In this example, `3AA5C34371567BD2` is the key ID you'll need to add to GitHub secrets.

Export your public key for distribution:

```bash
gpg --armor --export 3AA5C34371567BD2 > /var/www/repo/ssh-sync-repo.asc
chmod 644 /var/www/repo/ssh-sync-repo.asc
```

Make sure this file is accessible via your web server at `https://repo.sshsync.io/ssh-sync-repo.asc`

### 3. Web Server Configuration

#### Nginx Configuration

Create a new Nginx server block:

```bash
sudo nano /etc/nginx/sites-available/repo.sshsync.io
```

Add the following configuration:

```nginx
server {
    listen 80;
    server_name repo.sshsync.io;
    root /var/www/repo;
    autoindex on;

    location / {
        try_files $uri $uri/ =404;
    }

    location /debian {
        alias /var/www/repo/debian;
        autoindex on;
    }

    location /rpm {
        alias /var/www/repo/rpm;
        autoindex on;
    }
}
```

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/repo.sshsync.io /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

#### Apache Configuration

Create a new Apache virtual host:

```bash
sudo nano /etc/apache2/sites-available/repo.sshsync.io.conf
```

Add the following configuration:

```apache
<VirtualHost *:80>
    ServerName repo.sshsync.io
    DocumentRoot /var/www/repo

    <Directory /var/www/repo>
        Options +Indexes
        AllowOverride None
        Require all granted
    </Directory>

    ErrorLog ${APACHE_LOG_DIR}/repo.sshsync.io-error.log
    CustomLog ${APACHE_LOG_DIR}/repo.sshsync.io-access.log combined
</VirtualHost>
```

Enable the site:

```bash
sudo a2ensite repo.sshsync.io.conf
sudo a2enmod rewrite
sudo apache2ctl configtest
sudo systemctl reload apache2
```

### 4. SSL Configuration (Recommended)

We strongly recommend setting up SSL using Let's Encrypt:

```bash
sudo apt install -y certbot
# For Nginx
sudo apt install -y python3-certbot-nginx
sudo certbot --nginx -d repo.sshsync.io

# For Apache
sudo apt install -y python3-certbot-apache
sudo certbot --apache -d repo.sshsync.io
```

### 5. User Setup for GitHub Actions

Create a dedicated user for the GitHub Action to upload packages:

```bash
sudo useradd -m deployer
sudo mkdir -p /home/deployer/.ssh
```

Add your GitHub Actions public key to the authorized_keys file:

```bash
echo "YOUR_SSH_PUBLIC_KEY" | sudo tee -a /home/deployer/.ssh/authorized_keys
sudo chown -R deployer:deployer /home/deployer/.ssh
sudo chmod 700 /home/deployer/.ssh
sudo chmod 600 /home/deployer/.ssh/authorized_keys
```

Grant the deployer user access to the repository directories:

```bash
sudo setfacl -R -m u:deployer:rwx /var/www/repo
sudo setfacl -d -m u:deployer:rwx /var/www/repo
```

## Repository Management

### Debian Repository

After the GitHub Actions workflow uploads packages to your server, the repository metadata will be automatically updated. However, you can also update it manually:

```bash
cd /var/www/repo/debian
dpkg-scanpackages --multiversion . > Packages
gzip -k -f Packages
apt-ftparchive release . > Release
```

### RPM Repository

The RPM repository metadata will also be automatically updated by the GitHub Actions workflow. To update it manually:

```bash
cd /var/www/repo/rpm
createrepo_c .
```

## Client Configuration

### Adding the Debian Repository

Users can add the repository with the following commands:

```bash
echo "deb [trusted=yes] https://repo.sshsync.io/debian $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/ssh-sync.list
sudo apt update
```

### Adding the RPM Repository

Users can add the repository with the following commands:

```bash
cat <<EOF | sudo tee /etc/yum.repos.d/ssh-sync.repo
[ssh-sync]
name=SSH-Sync Repository
baseurl=https://repo.sshsync.io/rpm
enabled=1
gpgcheck=0
EOF
```

## Securing the Repository

The GitHub Actions workflow now signs all packages and repository metadata with GPG:

1. RPM packages are individually signed
2. The RPM repository metadata (repomd.xml) is signed
3. The Debian repository Release file is signed (creating InRelease and Release.gpg files)

Your users can securely verify the authenticity of packages by importing your GPG public key:

```bash
# For Debian/Ubuntu users
curl -fsSL https://repo.sshsync.io/ssh-sync-repo.asc | sudo gpg --dearmor -o /usr/share/keyrings/ssh-sync-archive-keyring.gpg

# For RPM-based distro users
sudo rpm --import https://repo.sshsync.io/ssh-sync-repo.asc
```

These commands should be added to your installation documentation.

## Troubleshooting

Common issues and their solutions:

- **403 Forbidden errors**: Check file and directory permissions
- **Packages not updating**: Ensure the repository metadata is regenerated after new uploads
- **SSH connection issues from GitHub Actions**: Verify the SSH key and connectivity