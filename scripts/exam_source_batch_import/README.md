# Exam Source Batch Import (External Python)

This tool scans a root folder where each subfolder is one paper:

- `scan_path/试卷1/`
- `scan_path/试卷2/`
- `scan_path/试卷3/`

It is designed for **step 1** only:

1. batch scan + collect page images
2. call backend analyze API (either simple analyze or save-review-record)
3. save per-paper artifacts for review

It does **not** auto-create papers by default.

## Folder Rules

- One subfolder = one paper.
- Only direct child directories under `scan_path` are processed.
- Supported image extensions: `.jpg`, `.jpeg`, `.png`, `.webp`.
- Images are sorted as pages by:
  1. first number in filename (natural order)
  2. filename lexicographic order
- Recommended naming:
  - `001.jpg`, `002.jpg`, ...
  - or `page_001.png`, `page_002.png`, ...

Optional per-paper metadata file:

- `scan_path/试卷1/meta.json`

Example:

```json
{
  "title": "2025 届高三物理二模",
  "term": "二模",
  "exam_year": 2025,
  "grade_label": "高三",
  "source_school": "XX中学",
  "source_region": "广州"
}
```

If omitted, folder name is used as title.

## Output Structure

Per run:

- `output/run_YYYYmmdd_HHMMSS/_summary.json`

Per paper:

- `output/run_.../<paper_key>/manifest.json`
- `output/run_.../<paper_key>/request_info.json`
- `output/run_.../<paper_key>/analyze_response.json` (if API call succeeds)
- `output/run_.../<paper_key>/error.json` (if failed)

## Install

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r scripts/exam_source_batch_import/requirements.txt
```

## Run

Dry-run (scan + manifest only):

```bash
python3 scripts/exam_source_batch_import/main.py \
  --scan-path "/path/to/scan_path" \
  --output-path "/path/to/output" \
  --dry-run
```

Call backend upload-analyze:

```bash
python3 scripts/exam_source_batch_import/main.py \
  --scan-path "/path/to/scan_path" \
  --output-path "/path/to/output" \
  --api-base "http://127.0.0.1:8080" \
  --admin-user "admin" \
  --admin-pass "your_password"

# default mode is `save-record`:
# POST /api/v1/admin/exam-source/import-records/upload-analyze
# (records become visible in admin for review/create)
```

Use old analyze-only mode:

```bash
python3 scripts/exam_source_batch_import/main.py \
  --scan-path "/path/to/scan_path" \
  --api-base "http://127.0.0.1:8080" \
  --admin-user "admin" \
  --admin-pass "your_password" \
  --mode analyze-only
```

Process only one subfolder:

```bash
python3 scripts/exam_source_batch_import/main.py \
  --scan-path "/path/to/scan_path" \
  --paper-dir "试卷2" \
  --api-base "http://127.0.0.1:8080" \
  --admin-user "admin" \
  --admin-pass "your_password"
```

## Notes

- This tool targets step-1 staging workflow.  
- Step-2 ("admin review then create paper") can consume these artifacts or a future staging API.
