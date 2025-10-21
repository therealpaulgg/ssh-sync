# Copy-Paste Prompt for ssh-sync-server Implementation

Copy and paste this entire section into a new Claude Code session working on the ssh-sync-server repository:

---

I need you to implement server-side support for the `known_hosts` synchronization feature that was just added to the ssh-sync client.

## Data Contract Change

The client now sends and expects a `known_hosts` field in the data API:

**Updated DataDto structure:**
```json
{
  "id": "uuid",
  "username": "string",
  "keys": [...],
  "ssh_config": [...],
  "known_hosts": "base64-encoded-bytes",  // NEW FIELD - encrypted binary data
  "machines": [...]
}
```

## Requirements

Implement the following changes to ssh-sync-server:

### 1. Database Migration
Add a `known_hosts` column to store the encrypted known_hosts data:
- Type: BYTEA (PostgreSQL) or equivalent binary type
- Nullable: YES (not all users have known_hosts files)
- Column name: `known_hosts`

### 2. Update Data Model/DTO
Add `KnownHosts []byte` (or language equivalent) field to the data DTO/model that corresponds to the client's DataDto.

### 3. Update POST /api/v1/data Endpoint
Modify the upload handler to:
- Extract the `"known_hosts"` field from the multipart form data
- Store the encrypted binary data in the database (do NOT decrypt server-side)
- Handle the case where known_hosts is not provided (it's optional)

### 4. Update GET /api/v1/data Endpoint
Modify the download handler to:
- Query the `known_hosts` column from the database
- Include it in the JSON response
- Handle NULL values gracefully (return null or empty array)

### 5. Ensure Backward Compatibility
- Old clients without known_hosts support should continue to work
- Missing known_hosts field should not cause errors

## Important Notes

- The data is **already encrypted client-side** - store it as-is
- known_hosts is **optional** - handle NULL/missing values
- The field contains **binary data** (encrypted bytes)
- Maintain backward compatibility with existing clients

## Implementation Steps

1. Create database migration for the new column
2. Update the data model/DTO
3. Modify the upload endpoint to accept and store known_hosts
4. Modify the download endpoint to retrieve and return known_hosts
5. Test upload and download flows
6. Commit with message: "Add known_hosts synchronization support"

Please implement these changes, ensuring the server properly stores and retrieves the known_hosts data for synchronization across client machines.
