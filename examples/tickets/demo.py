"""Populate a dedicated local demo through the application's authenticated API."""
import argparse
import json
import os
from pathlib import Path
import secrets
import sqlite3
import sys
from urllib.error import HTTPError
from urllib.parse import urlencode, urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener


class LocalRequests(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def main():
    args = argparse.ArgumentParser(description=__doc__)
    args.add_argument("--url", default="http://127.0.0.1:23100")
    args.add_argument("--credentials", required=True, type=Path)
    args.add_argument("--cleanup-sqlite", type=Path, metavar="DEMO_DB",
                      help="remove only this import's identified tickets from an offline local SQLite demo")
    options = args.parse_args()
    base = options.url.rstrip("/")
    target = urlsplit(base)
    if target.scheme not in ("http", "https") or target.hostname not in ("127.0.0.1", "::1") or target.path or target.username or target.query or target.fragment:
        args.error("Use a loopback origin for the demo.")
    file = options.credentials.resolve()
    if file.is_relative_to(Path(__file__).resolve().parents[2]):
        args.error("Store demo credentials outside the source directory.")
    manifest = file.with_name(file.name + ".tickets.json")
    if manifest.exists():
        batch = json.loads(manifest.read_text(encoding="utf-8"))
        if batch["origin"] != base:
            args.error("The manifest belongs to another preview origin.")
    else:
        if options.cleanup_sqlite:
            args.error("No import manifest found.")
        batch = {"origin": base, "tag": secrets.token_hex(6), "tickets": {}}

    def save_manifest():
        tmp = manifest.with_suffix(".tmp")
        tmp.write_text(json.dumps(batch, indent=2), encoding="utf-8")
        tmp.replace(manifest)

    if options.cleanup_sqlite:
        db = sqlite3.connect(options.cleanup_sqlite.resolve().as_uri() + "?mode=rw", uri=True)
        with db:
            db.execute("BEGIN IMMEDIATE")
            removed = 0
            for title, record in batch["tickets"].items():
                if not title.startswith("[Demo " + batch["tag"] + "] "):
                    raise ValueError("Unexpected ticket marker in the manifest")
                if "version" not in record:
                    continue
                identity = (record["id"], record["owner_id"], record["created_at"], title, record["version"])
                match = db.execute("SELECT id FROM example_tickets WHERE id=? AND owner_id=? AND created_at=? AND title=? AND version=?", identity).fetchone()
                if match:
                    db.execute("DELETE FROM example_ticket_entries WHERE ticket_id=?", match)
                    db.execute("DELETE FROM example_tickets WHERE id=?", match)
                    removed += 1
        db.close()
        print(f"Removed {removed} matching demo tickets. Accounts and audit records are retained.")
        return
    opener = build_opener(LocalRequests())

    def api(method, route, payload=None, token=None, proof=None, multipart=False, attachment=False):
        headers = {}
        body = None
        if multipart:
            boundary = secrets.token_hex(16)
            parts = [f'--{boundary}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n'.encode() for k, v in payload.items()]
            if attachment:
                parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="attachment"; filename="checklist.txt"\r\nContent-Type: text/plain\r\n\r\nVerify the cable and connector.\r\n'.encode())
            body = b"".join(parts) + f"--{boundary}--\r\n".encode()
            headers["Content-Type"] = "multipart/form-data; boundary=" + boundary
        elif payload is not None:
            body = json.dumps(payload).encode()
            headers["Content-Type"] = "application/json"
        if token:
            headers["Authorization"] = "Bearer " + token
        if proof:
            headers["X-Security-Proof"] = proof
        try:
            with opener.open(Request(base + route, data=body, headers=headers, method=method), timeout=20) as res:
                result = json.load(res)
        except HTTPError as exc:
            raise RuntimeError(f"{method} {route}: HTTP {exc.code}") from None
        if not result.get("success"):
            raise RuntimeError(f"{method} {route}: {result.get('message','request failed')}")
        return result.get("data")

    setup = api("GET", "/api/setup")
    if file.exists():
        accounts = json.loads(file.read_text(encoding="utf-8"))
    else:
        if setup["status"]:
            args.error("This instance has been initialized. Use its original demo credentials file or start with a fresh demo database.")
        file.parent.mkdir(parents=True, exist_ok=True)
        accounts = {name: {"username": name, "password": secrets.token_urlsafe(24)} for name in ("root", "admin", "user")}
        with os.fdopen(os.open(file, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600), "w", encoding="utf-8") as out:
            json.dump(accounts, out, indent=2)
    if not setup["status"]:
        api("POST", "/api/setup", {**accounts["root"], "confirmPassword": accounts["root"]["password"]})
    root = api("POST", "/api/user/login", accounts["root"])["access_token"]
    for name, role in (("admin", 10), ("user", 1)):
        matches = api("GET", "/api/user/search?keyword=" + name, token=root)["items"]
        if not any(u["username"] == name for u in matches):
            payload = {**accounts[name], "role": role, "display_name": name.capitalize()}
            proof = None
            if role == 10:
                payload["admin_permissions"] = {"ticket":{"read":True,"write":True}}
                proof = api("POST", "/api/verify", {
                    "scope":"admin.user.create","context":{"role":role},"method":"password","password":accounts["root"]["password"]
                },root)["proof_token"]
            api("POST", "/api/user/", payload, root, proof)
    admin = api("POST", "/api/user/login", accounts["admin"])["access_token"]
    user = api("POST", "/api/user/login", accounts["user"])["access_token"]
    save_manifest()
    samples = [
        ("Meeting room display", "Please check the display cable in meeting room B.", user, admin, 0),
        ("Review the access request", "Please review read access for the new team member.", user, admin, 1),
        ("Replace the keyboard", "The replacement keyboard needs to be delivered.", user, admin, 2),
        ("Archive cabinet lock", "The archive cabinet lock needs repair.", admin, root, 2),
        ("Printer maintenance", "Please schedule maintenance for the shared printer.", root, admin, 1),
    ]
    for room in range(1, 10):
        samples.append((f"Check workstation {room}", f"Check the network connection at desk {room}.", user, admin, 0))

    for index, (subject, description, owner, handler, steps) in enumerate(samples):
        title = f"[Demo {batch['tag']}] {subject}"
        items = api("GET", "/api/tickets?" + urlencode({"search": title}), token=owner)["items"]
        row = next((item for item in items if item["title"] == title), None)
        if row is None:
            row = api("POST", "/api/tickets", {"title": title, "description": description},
                      owner, multipart=True, attachment=index in (0, 2))
        route = f"/api/manage/tickets/{row['id']}"
        previous = batch["tickets"].get(title)
        if previous and previous["id"] == row["id"] and previous.get("ready"):
            continue
        record = {key: row[key] for key in ("id", "created_at", "owner_id")}
        batch["tickets"][title] = record
        save_manifest()
        if steps and row["status"] == "pending":
            api("PATCH", route + "/status", {"status": "in_progress", "version": row["version"]},handler)
        if steps:
            detail = api("GET", route, token=handler)
            if detail["status"] == "in_progress" and not any(e["action"] == "reply" for e in detail["entries"]):
                api("POST", route + "/replies", {"body": "The request has been checked and the work is scheduled.", "version":detail["version"]},handler)
            if steps == 2:
                detail = api("GET", route, token=handler)
                if detail["status"] == "in_progress":
                    api("PATCH", route + "/status", {"status":"completed","version":detail["version"]},handler)
        record["version"] = api("GET", f"/api/tickets/{row['id']}", token=owner)["version"]
        record["ready"] = True
        save_manifest()
    print(f"Demo ready at {base}/tickets. Accounts: root, admin, user. Credentials: {file}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError) as error:
        sys.exit(str(error))
