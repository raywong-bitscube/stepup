#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
import json
import pathlib
import re
import sys
from dataclasses import dataclass
from typing import Any, Iterable

import requests

IMAGE_EXTS = {".jpg", ".jpeg", ".png", ".webp"}
NUMBER_RE = re.compile(r"(\d+)")


@dataclass
class PaperJob:
    name: str
    path: pathlib.Path
    images: list[pathlib.Path]
    meta: dict[str, Any]


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Batch scan exam folders and call upload-analyze.")
    p.add_argument("--scan-path", required=True, help="Root folder; direct subdirs are papers.")
    p.add_argument("--output-path", default="output", help="Output root for run artifacts.")
    p.add_argument("--paper-dir", default="", help="Only process this one subfolder name.")
    p.add_argument("--dry-run", action="store_true", help="Only scan and emit manifests.")
    p.add_argument("--api-base", default="", help="Backend base URL, e.g. http://127.0.0.1:8080")
    p.add_argument("--admin-user", default="admin")
    p.add_argument("--admin-pass", default="")
    p.add_argument(
        "--mode",
        choices=("analyze-only", "save-record"),
        default="save-record",
        help="analyze-only => call /papers/upload-analyze; save-record => call /import-records/upload-analyze",
    )
    return p.parse_args()


def first_number_key(s: str) -> tuple[int, str]:
    m = NUMBER_RE.search(s)
    if not m:
        return (10**9, s.lower())
    return (int(m.group(1)), s.lower())


def sort_images(paths: Iterable[pathlib.Path]) -> list[pathlib.Path]:
    return sorted(paths, key=lambda x: first_number_key(x.name))


def read_meta(paper_dir: pathlib.Path) -> dict[str, Any]:
    meta_path = paper_dir / "meta.json"
    if not meta_path.exists():
        return {}
    try:
        return json.loads(meta_path.read_text(encoding="utf-8"))
    except Exception:
        return {}


def collect_jobs(scan_path: pathlib.Path, paper_dir_name: str) -> list[PaperJob]:
    if not scan_path.exists() or not scan_path.is_dir():
        raise ValueError(f"scan path not found or not directory: {scan_path}")
    subdirs = [x for x in scan_path.iterdir() if x.is_dir()]
    subdirs.sort(key=lambda x: x.name.lower())
    if paper_dir_name:
        subdirs = [x for x in subdirs if x.name == paper_dir_name]
    jobs: list[PaperJob] = []
    for d in subdirs:
        imgs = [f for f in d.iterdir() if f.is_file() and f.suffix.lower() in IMAGE_EXTS]
        imgs = sort_images(imgs)
        if not imgs:
            continue
        jobs.append(PaperJob(name=d.name, path=d, images=imgs, meta=read_meta(d)))
    return jobs


def ensure_dir(p: pathlib.Path) -> None:
    p.mkdir(parents=True, exist_ok=True)


def write_json(path: pathlib.Path, data: Any) -> None:
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")


def login(api_base: str, user: str, pwd: str) -> str:
    if not api_base:
        raise ValueError("api-base is required when not dry-run")
    if not pwd:
        raise ValueError("admin-pass is required when not dry-run")
    url = api_base.rstrip("/") + "/api/v1/admin/auth/login"
    resp = requests.post(url, json={"username": user, "password": pwd}, timeout=30)
    resp.raise_for_status()
    data = resp.json()
    token = str(data.get("token", "")).strip()
    if not token:
        raise RuntimeError("login success but token missing")
    return token


def build_title(job: PaperJob) -> str:
    title = str(job.meta.get("title", "")).strip()
    return title or job.name


def call_upload_analyze(api_base: str, token: str, job: PaperJob, mode: str) -> tuple[int, dict[str, Any], str]:
    if mode == "save-record":
        endpoint = "/api/v1/admin/exam-source/import-records/upload-analyze"
    else:
        endpoint = "/api/v1/admin/exam-source/papers/upload-analyze"
    url = api_base.rstrip("/") + endpoint
    files = []
    handles = []
    try:
        for img in job.images:
            fh = img.open("rb")
            handles.append(fh)
            files.append(("images", (img.name, fh, "application/octet-stream")))
        data = {"title": build_title(job), "source_dir": str(job.path)}
        headers = {"Authorization": f"Bearer {token}"}
        resp = requests.post(url, data=data, files=files, headers=headers, timeout=300)
        try:
            payload = resp.json()
        except Exception:
            payload = {"raw": resp.text}
        return resp.status_code, payload, endpoint
    finally:
        for h in handles:
            h.close()


def run() -> int:
    args = parse_args()
    scan_path = pathlib.Path(args.scan_path).expanduser().resolve()
    out_root = pathlib.Path(args.output_path).expanduser().resolve()
    run_tag = dt.datetime.now().strftime("run_%Y%m%d_%H%M%S")
    run_dir = out_root / run_tag
    ensure_dir(run_dir)

    jobs = collect_jobs(scan_path, args.paper_dir.strip())
    summary = {
        "scan_path": str(scan_path),
        "run_dir": str(run_dir),
        "total_dirs": len(jobs),
        "dry_run": bool(args.dry_run),
        "items": [],
    }

    token = ""
    if not args.dry_run:
        token = login(args.api_base.strip(), args.admin_user.strip(), args.admin_pass)

    for idx, job in enumerate(jobs, start=1):
        paper_key = f"{idx:03d}_{job.name}"
        paper_out = run_dir / paper_key
        ensure_dir(paper_out)

        manifest = {
            "paper_name": job.name,
            "paper_path": str(job.path),
            "title_hint": build_title(job),
            "meta": job.meta,
            "images": [x.name for x in job.images],
        }
        write_json(paper_out / "manifest.json", manifest)

        item = {"paper_key": paper_key, "paper_name": job.name, "status": "ok", "error": ""}
        if args.dry_run:
            summary["items"].append(item)
            continue

        request_info = {
            "api_base": args.api_base.strip(),
            "endpoint": "",
            "title_hint": build_title(job),
            "images_count": len(job.images),
            "mode": args.mode,
        }
        write_json(paper_out / "request_info.json", request_info)
        try:
            code, payload, endpoint = call_upload_analyze(args.api_base.strip(), token, job, args.mode)
            request_info["endpoint"] = endpoint
            write_json(paper_out / "request_info.json", request_info)
            write_json(paper_out / "analyze_response.json", {"status_code": code, "payload": payload})
            if code < 200 or code >= 300:
                item["status"] = "failed"
                item["error"] = f"http {code}"
        except Exception as exc:
            item["status"] = "failed"
            item["error"] = str(exc)
            write_json(paper_out / "error.json", {"error": str(exc)})
        summary["items"].append(item)

    summary["ok_count"] = sum(1 for x in summary["items"] if x["status"] == "ok")
    summary["failed_count"] = sum(1 for x in summary["items"] if x["status"] != "ok")
    write_json(run_dir / "_summary.json", summary)
    print(f"done: {run_dir}")
    print(f"ok={summary['ok_count']} failed={summary['failed_count']}")
    return 0 if summary["failed_count"] == 0 else 2


if __name__ == "__main__":
    sys.exit(run())

