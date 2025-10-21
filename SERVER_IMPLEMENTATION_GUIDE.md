# Server-Side Implementation Guide: known_hosts Sync Support

## Context

The ssh-sync client has been updated to support synchronizing the `known_hosts` file across machines. The client now:

1. Reads and encrypts `~/.ssh/known_hosts` during upload
2. Sends it as a `known_hosts` form field in the multipart upload
3. Expects to receive `known_hosts` data in the response when downloading
4. Decrypts and writes the known_hosts file during download

## Client-Side Changes (Already Implemented)

### DataDto Structure
```go
type DataDto struct {
    ID         uuid.UUID      `json:"id"`
    Username   string         `json:"username"`
    Keys       []KeyDto       `json:"keys"`
    SshConfig  []SshConfigDto `json:"ssh_config"`
    KnownHosts []byte         `json:"known_hosts"`  // NEW FIELD
    Machines   []MachineDto   `json:"machines"`
}
```

### Upload Behavior
- Client reads `~/.ssh/known_hosts` file (if exists)
- Encrypts it with the user's master key (AES-GCM)
- Sends as multipart form field named `"known_hosts"`
- Only sends if the file exists (otherwise field is omitted)

### Download Behavior
- Client expects `known_hosts` field in JSON response
- Decrypts the data
- Writes to `~/.ssh/known_hosts` with conflict resolution
- If `known_hosts` is empty or null, no file is written

## Required Server-Side Implementation

You need to implement server-side support for storing and retrieving the known_hosts data. Follow these steps:

### Step 1: Update Database Schema

Add a `known_hosts` column to the table that stores user SSH data. This should:
- Store binary data (bytea/blob type)
- Be nullable (not all users will have known_hosts)
- Be associated with the user's data record

**Expected table**: Look for the table that stores SSH keys and config data (likely named something like `data`, `user_data`, or `ssh_data`)

### Step 2: Update Data Model/DTO

Find the server-side equivalent of `DataDto` and add the `known_hosts` field:
- Field name: `KnownHosts` or `known_hosts`
- Type: `[]byte`, `bytes`, or equivalent binary type
- JSON tag: `json:"known_hosts"`
- Should match the client's DataDto structure

### Step 3: Update Upload Endpoint

Find the endpoint that handles SSH data uploads (likely `POST /api/v1/data`). Update it to:

1. **Extract known_hosts from multipart form**:
   - Read the `"known_hosts"` form field
   - Handle the case where it doesn't exist (optional field)
   - Store the encrypted binary data as-is (don't decrypt server-side)

2. **Save to database**:
   - Include known_hosts in the INSERT/UPDATE query
   - Handle NULL when known_hosts is not provided

**Example pseudocode**:
```
knownHostsData := extractFormField("known_hosts") // may be nil/empty
if knownHostsData != nil {
    userData.KnownHosts = knownHostsData
}
saveToDatabase(userData)
```

### Step 4: Update Download Endpoint

Find the endpoint that serves SSH data (likely `GET /api/v1/data`). Update it to:

1. **Query known_hosts from database**:
   - Include known_hosts column in SELECT query
   - Handle NULL values

2. **Include in JSON response**:
   - Add known_hosts to the response DTO
   - Empty/null values should serialize as null or empty array in JSON

**Example pseudocode**:
```
userData := queryUserData(userId)
response := DataDto{
    Keys: userData.Keys,
    SshConfig: userData.SshConfig,
    KnownHosts: userData.KnownHosts, // may be nil
    ...
}
return json(response)
```

### Step 5: Testing Considerations

After implementation, test:
1. Upload with known_hosts → should save to database
2. Upload without known_hosts → should save NULL/empty
3. Download with known_hosts → should return in JSON
4. Download without known_hosts → should return null/empty
5. Backward compatibility → old clients should not break

### Step 6: Migration

Create a database migration to add the new column:
- Column name: `known_hosts`
- Type: BYTEA (PostgreSQL), BLOB (MySQL), or equivalent
- Nullable: YES
- Default: NULL

**Example PostgreSQL migration**:
```sql
ALTER TABLE user_data ADD COLUMN known_hosts BYTEA NULL;
```

## Important Notes

1. **Don't decrypt server-side**: The server stores encrypted data only. The client handles all encryption/decryption.

2. **Backward compatibility**: Ensure old clients (without known_hosts support) can still upload/download without errors.

3. **Optional field**: known_hosts is optional - handle NULL/missing values gracefully.

4. **Binary data**: known_hosts is binary (encrypted) data, not text. Use appropriate binary types and encoding.

5. **Size limits**: known_hosts files can be large. Ensure your upload size limits accommodate them (typically a few KB, but can grow to hundreds of KB).

## Expected File Locations

Look for these files in the ssh-sync-server repository:
- Database models: `/models/`, `/pkg/models/`, `/internal/models/`
- DTOs: `/dto/`, `/pkg/dto/`, `/internal/dto/`
- Upload handler: `/handlers/`, `/api/`, `/routes/`
- Download handler: `/handlers/`, `/api/`, `/routes/`
- Migrations: `/migrations/`, `/db/migrations/`

## Verification

After implementation, verify:
- [ ] Database schema updated with known_hosts column
- [ ] DTO/model includes KnownHosts field
- [ ] Upload endpoint extracts and saves known_hosts
- [ ] Download endpoint retrieves and returns known_hosts
- [ ] Migration file created
- [ ] Tests updated (if applicable)
- [ ] No breaking changes for existing clients

## Commit Message Template

```
Add known_hosts synchronization support

- Add known_hosts column to database schema
- Update DataDto to include KnownHosts field
- Implement known_hosts upload in POST /api/v1/data endpoint
- Implement known_hosts download in GET /api/v1/data endpoint
- Create migration for known_hosts column
- Maintain backward compatibility with older clients

This complements the client-side known_hosts sync feature,
allowing users to sync their SSH known_hosts file across machines.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```
