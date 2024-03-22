# SSH Sync

A CLI application to sync SSH keys along with a SSH configuration between machines on-demand.

Also part of this project:

[https://github.com/therealpaulgg/ssh-sync-server](https://github.com/therealpaulgg/ssh-sync-server)  
[https://github.com/therealpaulgg/ssh-sync-db](https://github.com/therealpaulgg/ssh-sync-db)

## Diagram

This diagram depicts the following:

1. Initial user setup with server
2. Sequence diagram illustrating a user configuring a new PC
3. Authenticated download request
4. Authenticated upload request

![svg-of-ssh-sync-architecture](https://raw.githubusercontent.com/therealpaulgg/ssh-sync/main/docs/diagrams.svg)

## High-Level Concept

You have a computer which has SSH keys and configuration on them, and on a secondary machine you would like these keys so you are able to access your servers from another machine. This often involves copy-pasting files and also rewriting an SSH configuration file because file paths change, or the operating system changes. This can be quite tedious. This program aims to replace the manual steps involved by allowing a secure method of transferring keys and keeping them all synced in a remote server.

## Why not P2P?

P2P was considered (and may still be done at a later  date) but ultimately dismissed because every key sync request would have to involve two machines, which would get tedious quickly. The main idea is a transaction involving two machines at the same time would only have to be done once (to allow a new machine access to a user's keys). After this point, the new machine can communicate to the server freely, uploading and downloading new keys. A P2P implementation would mean that a machine would have to do a 'handshake' with another machine each time. There would also be no central source to keep key data synchronized.

# Technical Details (Crypto)

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

Each user will have a 'Master Key'. This will be a unique symmetric key. Each client will have the master key on their machine, encrypted with their public key. For example, on computer A there would be `E_pubA(master_key)` and on computer B `E_pubB(master_key)`. All of the user's data (SSH keys) will be stored on the server in an encrypted AES 256 GCM format. This data can only be decrypted with the master symmetric key, which can only be decrypted by the user using one of their user/machine keypairs.

NOTE: the master key was previously stored on the server, but this was found to be redundant and unnecessary. It is practically just as secure to keep the master key on each machine (encrypted or even unencrypted theoretically). If someone has the public/private keypair, they would be able to easily retrieve the master key for that machine from the server anyways.

#### Upload

Whenever the user wants to upload new data, the server will send the encrypted master key. The user will then decrypt the master key, and send encrypted keys to the server. 

Server sends: `E_pubMachine(master_key)`

Machine decrypts master key, `D_privMachine(E_pubMachine(master_key))`

Machine encrypts keys with `master_key` and also signs the data.

`E_privMachine(E_masterKey(plaintext))`

Server receives this data and validates the signature:

`D_pubMachine(ciphertext)`

The server will then store this `ciphertext`.

#### Download

Whenever the user wants to download data, the server will send all the user's data in encrypted format, as well as the encrypted master key.

The user, once it receives the data, will decrypt the master key using their private key, and then decrypt the ciphertext.

Server sends: `E_pubMachine(ciphertext), E_pubMachine(master_key)`

Machine decrypts master key, `D_privMachine(E_pubMachine(master_key))`

Machine decrypts ciphertext.

`D_masterKey(D_privMachine(ciphertext))`

## Adding New Machines

Adding a new machine will require one of the user's other machines. The new machine will make a request to the server to be added. The server will then respond with some sort of challenge, requesting that the user enters a phrase on one of their old machines. Once the user enters the phrase on their old machine, the server will then allow them to upload a public keypair so they can do communication with the server. The old machine will need to be involved a little longer so that it can decrypt the master key, and upload a new master key using this new machine's public key. Here is the full process laid out:

Machine B requests to be added to allowed clients  
Server gives Machine B a challenge phrase which must be entered on Machine A  
Challenge phrase is entered on Machine A.  Machine A awaits response from server (Machine B's public key)  
Machine B generates keypair and uploads public key to server.  
Server saves B's public key and then sends `E_aPub(master_key)` & B's public key to Machine A.  
Machine A does `D_aPriv(enc_master_key)`, and then sends `E_bPub(master_key)` to the server.  
Server then saves this new encrypted master key.

# Other Technical Details

## SSH Config Parsing

Part of the syncing process will be where the program parses a user's SSH config file, and then sends the parsed format over to the server. Other machines, when syncing, will be able to have new config files generated for them based on what is in the server (and the CLI will handle changing user directories & OS changes).

## Data Conflicts

TODO: after parsing SSH config, if there are duplicate entries, attempt to merge, but with conflicts, ask user how to resolve.

TODO: what if duplicate keys get uploaded? ask user to replace/skip

# P2P Concept

If P2P was to be implemented, a lot of the crypto needed for the server implementation would be unnecessary. This is how a request would probably go:

Machine B wants Machine A's keys and config.  
Machine A challenges Machine B with a phrase.  
Machine B responds to challenge, and sends its public key.  
Assuming challenge is passed, Machine A encrypts its data using EC-DH-A256GCM (Machine B public key) and sends it over to Machine B.  
Machine B would receive the data, decrypt it, and the program would manage things as necessary.

All the other functionality of the program (i.e SSH config parser) would remain the same.

