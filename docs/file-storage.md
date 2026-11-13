# Private report attachment storage

One optional attachment/report, stored as an opaque relative generated key in the existing reports.attachment_url column. No binary in PostgreSQL, no extra attachment metadata table, static server or public download route.

## Technical defaults

| Setting | Default |
| --- | --- |
| UPLOAD_DIR | ./var/private/attachments, resolved absolute on server |
| UPLOAD_MAX_FILE_BYTES | 5242880 (5 MiB) |
| UPLOAD_MAX_REQUEST_BYTES | 6291456 (6 MiB, >=file+64 KiB) |
| UPLOAD_ALLOWED_TYPES | image/jpeg,image/png,application/pdf |
| UPLOAD_MAX_FILES | 1; 0 disables uploads; >1 rejected due single attachment schema |
| REPORT_UPLOAD_CONCURRENCY | 2 (1..16) |
| REPORT_REQUEST_TIMEOUT | 10s, less than write timeout |

Byte ceilings accept 1..32 MiB, with request overhead relationship enforced. Allowed types can be restricted to a subset of the three supported validators, never expanded to executable formats. Next server uses UPLOAD_MAX_REQUEST_BYTES too; align it with Go. GET /api/v1/public/reports/options exposes only this non-sensitive technical upload policy for frontend validation. These defaults are technical choices, not added business rules.

UPLOAD_DIR must be outside frontend/public, .next, static or any configured web root; known public path components are rejected by config. Operator must keep custom web-server aliases away from this path too. Directory and staging are real private directories with permissions0700; insecure existing permissions or symlink directories cause startup error instead of silently changing an existing directory. Files use0600 and random256-bit hex names, with canonical .jpg/.png/.pdf extension. Original filenames never become storage paths. Go os.Root confines filesystem operations.

## Streaming validation

Global Phase 0 body limit stays1MiB on JSON/admin endpoints. Only exact POST public/reports uses its own bounded streaming parser. No ParseMultipartForm or ReadAll of whole request; maximum six parts (five scalar fields plus file), duplicate/unknown parts rejected, scalar values bounded separately. Actual Content-Disposition filename is checked before Go's FileName basename sanitization so traversal attempts are not silently accepted. Both slash variants, NUL, .., empty/long names are rejected.

File copy writes at most max_file+1 bytes to private staging, checks empty/oversize, MIME header (when provided), actual sniffed signature and extension consistency. JPEG/PNG undergo DecodeConfig and full decode, capped at20 million pixels; end marker rejects truncated/trailing data. PDF undergoes strict pdfcpu structural validation with stateless configuration, offline mode, URL checking/optimization disabled, context cancellation and configured parser resource ceilings. No external network/font/config directory access is needed for validation. This parser dependency is justified by structural checks beyond signature-only acceptance.

PDF parser limits include object count5000, xref entries50000, recursion100, object stream count5000, prolog1MiB, object/encoded stream16MiB, per-decode32MiB and image limits. These bound parser operations; they are not a whole-process memory sandbox. Strict validation can reject nonconforming PDFs that permissive viewers open. It is not malware scanning or content disarm; files remain private and are never executed/rendered by this phase.

## Publish and cleanup

Stage -> validate -> sync/close -> begin report transaction -> insert attachment key -> atomic rename on same filesystem -> commit. The key is never committed before file publication succeeds. No request exposes an absolute path or generated key. The storage interface Stage/Publish/Remove can later be implemented by a private object store without moving business logic into handlers.

Known rollback/validation/storage/database failure removes staging and any final file using a fresh bounded cleanup context even after request cancellation. Cleanup failure logs a generic reconciliation-needed event without path/contact. Test fault injection publishes a real file then fails acknowledgment: PostgreSQL report/counter roll back and final file is removed.

## Crash and ambiguous commit recovery

Filesystem rename and PostgreSQL commit cannot form one distributed transaction. Crash after rename but before commit can leave an orphan; crash during staging can leave staging files. File data is fsynced before publication; directory rename durability across sudden power loss is not guaranteed by a directory fsync in this implementation.

On network/connection failure during COMMIT, outcome may be uncertain. Application deliberately retains any published file and returns SUBMISSION_UNCERTAIN rather than deleting a potentially committed attachment. Definite PostgreSQL/transaction rollback errors follow normal cleanup. Before cleanup of old files, compare keys against reports.attachment_url using a trusted operator process and an age grace period; never delete newly published files based on an immediate negative lookup while a commit may still be in flight. No automatic destructive orphan cleanup job is included. Back up private files and database together when preparing deployment; deployment is outside scope.
