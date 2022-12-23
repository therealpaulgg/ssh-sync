# SSH Sync

A CLI application to sync SSH keys along with a SSH configuration between machines on-demand.

## High-Level Concept

You have a computer which has SSH keys and configuration on them, and on a secondary machine you would like these keys so you are able to access your servers from another machine. This often involves copy-pasting files and also rewriting an SSH configuration file because file paths change, or the operating system changes. This can be quite tedious. This program aims to replace the manual steps involved by allowing a secure method of transferring keys and keeping them all synced in a remote server.

## Why not P2P?

P2P was considered (and may still be done at a later  date) but ultimately dismissed because every key sync request would have to involve two machines, which would get tedious quickly. The main idea is a transaction involving two machines at the same time would only have to be done once (to allow a new machine access to a user's keys). After this point, the new machine can communicate to the server freely, uploading and downloading new keys. A P2P implementation would mean that a machine would have to do a 'handshake' with another machine each time. There would also be no central source to keep key data synchronized.

# Technical Details

## Initial Setup

On initial setup, a user will be asked to provide a username and name for the current machine. A ECDSA keypair will also be created, and the user's public key will be uploaded to the server, corresponding to the username and machine name. From this point on, the user will be able to communicate to the server and upload/download keys, as well as config data.

## Communication to Server

### JWT for Requests

All requests to the server (besides initial setup) will be done with a JWT. This JWT will be crafted on the client-side. It will be crafted using the ES512 algorithm, using the user's private key. The token will then be sent to the server, and validated with the user's public key. The crafted token needs to contain the following:

- Username
- Machine Name

the server will then look up the public key corresponding to the username and machine name provided, and attempt to validate the signature. It will be impossible for someone to forge a JWT for a particular user/machine pair because it would require the private key.

For example, if Eve crafts a JWT that says `{"username": "Alice", "machineName": "my-computer"}`, the server will attempt to verify the JWT using Alice's 'my-computer' keypair. Eve cannot impersonate Alice unless she gets her hands on Alice's private key.

## Storing Keys On The Server Securely

### Master Key

Each user will have a 'Master Key'. This will be a unique symmetric key. This symmetric key will be stored on the server, one copy for each keypair. All of the user's data will be stored on the server in an encrypted AES 256 GCM format. This data can only be decrypted with the master symmetric key, which can only be decrypted by the user using one of their user/machine keypairs.

#### Upload

Whenever the user wants to upload new data, the server will send the encrypted master key. The user will then decrypt the master key, and send encrypted keys to the server. 

Server sends: `encrypted_master_key`

Machine decrypts master key, `master_key`

Machine encrypts keys with `master_key` and also signs the data.

`E_privMachine(E_masterKey(plaintext))`

Server receives this data and validates the signature:

`D_pubMachine(ciphertext)`

The server will then store this `ciphertext`.

#### Download

Whenever the user wants to download data, the server will send all the user's data in encrypted format, as well as the encrypted master key.

The user, once it receives the data, will decrypt the master key using their private key, and then decrypt the ciphertext.

Server sends: `E_pubMachine(ciphertext), encrypted_master_key`

Machine decrypts master key, `master_key`

Machine decrypts ciphertext.

`D_masterKey(D_privMachine(ciphertext))`

## Data Conflicts

TODO: after parsing SSH config, if there are duplicate entries, attempt to merge, but with conflicts, ask user how to resolve.

TODO: what if duplicate keys get uploaded? ask user to replace/skip