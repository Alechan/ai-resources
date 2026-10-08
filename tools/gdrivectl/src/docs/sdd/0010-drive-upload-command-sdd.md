# SDD-0010: Drive File Upload Command

Status: Draft  
Version: 0.1  
Last updated: 2026-03-10

## 1. Purpose

Add first-class file upload support to `gdrivectl` so users can create Drive files from local paths via CLI.

## 2. Scope

### In scope

- New CLI command: `upload`
- Local-file upload to Google Drive using Drive API `files.create`
- Shared-drive-compatible upload flags (`supportsAllDrives=true`)
- JSON success output and existing categorized error semantics

### Out of scope

- Resumable/chunked uploads for very large files
- Folder creation and recursive directory upload
- Update/overwrite existing files by id

## 3. Inner Workings

1. `cmd/gdrivectl` parses `upload` flags and validates required `--path`.
2. `UploadService`:
   - validates path exists and is a file
   - reads file bytes from disk
   - derives default `name` from basename when `--name` is omitted
   - derives default MIME type from extension (fallback to content sniff, then `application/octet-stream`)
   - resolves access token via existing gcloud token provider
3. `DriveHTTPClient.Upload` sends multipart request to:
   - `POST https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&supportsAllDrives=true`
   - part 1: JSON metadata (`name`, optional `parents`)
   - part 2: media bytes with resolved MIME type
4. API response is returned as JSON map and printed by the shared output writer.
5. Failures remain mapped to existing categories:
   - validation/config -> exit `2`
   - auth/scope -> exit `3`
   - network -> exit `4`
   - api -> exit `5`

## 4. CLI Interface

Command:

```bash
gdrivectl upload --path <local_file> [--name <drive_name>] [--parent-id <folder_id>] [--mime <mime_type>] [--fields <projection>] [--json]
```

Flags:

- `--path` (required): local source file to upload
- `--name` (optional): destination Drive file name (default: basename from `--path`)
- `--parent-id` (optional): destination folder id
- `--mime` (optional): explicit media MIME type
- `--fields` (optional): Drive fields projection (default: `id,name,mimeType,webViewLink,parents`)

Success output:

- JSON object containing uploaded file metadata (`id`, `name`, etc. based on `--fields`)

## 5. Tasks

1. Add `UploadRequest` model and `DriveClient.Upload` interface method.
2. Implement multipart upload in `internal/googleapi/drive_client.go`.
3. Add `UploadService` with path validation, MIME/name defaults, and token resolution.
4. Add `cmd/gdrivectl/cmd_upload.go` and wire command in `root.go`.
5. Register upload service in `internal/app/app.go`.
6. Add tests:
   - multipart request contract + status mapping in `drive_client_test.go`
   - upload service behavior in `upload_service_test.go`
   - root required-flag validation for `upload`
7. Update docs (`README`, `TEST_PLAN`, SDD index, skill command map).

## 6. Acceptance Criteria

- `gdrivectl upload --path <file> --json` uploads successfully and returns JSON metadata.
- Missing/invalid `--path` returns validation error and exit `2`.
- Upload HTTP 401/403/5xx map to auth/scope/api categories consistently.
- Existing commands and tests remain green (`go test ./...`).
