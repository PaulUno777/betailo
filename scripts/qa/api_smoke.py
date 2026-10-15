"""End-to-end API smoke test over every profile (guest, buyer, farmer, admin).

Walks the real API through each profile's flows — browsing and search,
signup, opening a farm and getting it approved by staff (proof upload,
rejection and resubmission, the per-account farm limit), the sell wizard
with a photo upload,
Mobile Money checkout, the waiting list, delivery and escrow, reviews,
disputes, moderation and suspension — including the refusals (IDOR
attempts, invalid input, forbidden roles) and the error codes clients
translate. Each check prints PASS/FAIL; the exit code is the number of
failures (capped at 1), so CI fails on any.

Creates fresh accounts on every run, so it's repeatable against any
database. Needs a super admin account (nobody registers into the staff;
the API creates it at startup from SUPER_ADMIN_EMAIL/PASSWORD), and
expects MAX_FARMS_PER_ACCOUNT=1 and FARM_AUTO_APPROVE=false:

    QA_API=http://localhost:8080          (default)
    QA_ADMIN_EMAIL=... QA_ADMIN_PASSWORD=...
    QA_RESULTS=path/to/results.json       (optional)

Run it with scripts/qa/run_smoke.sh, which starts the stack with those
settings.
"""
import json
import os, os, sys, time, uuid, urllib.request, urllib.error, http.cookiejar, datetime

BASE = os.environ.get("QA_API", "http://localhost:8080").rstrip("/")
CREDS = {"ADMIN": os.environ["QA_ADMIN_EMAIL"], "PW": os.environ["QA_ADMIN_PASSWORD"]}
# Registration is rate-limited per IP; a short pause keeps a run (and a
# rerun right after) under it.
REGISTER_PAUSE = float(os.environ.get("QA_REGISTER_PAUSE", "2.5"))
RESULTS = []


class Client:
    def __init__(self):
        self.jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
        self.token = None

    def call(self, method, path, body=None, token=True, headers=None):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(BASE + path, data=data, method=method)
        req.add_header("Content-Type", "application/json")
        req.add_header("Origin", "http://localhost:4200")
        if token and self.token:
            req.add_header("Authorization", "Bearer " + self.token)
        for k, v in (headers or {}).items():
            req.add_header(k, v)
        # Chrome sends Secure cookies to http://localhost; urllib doesn't.
        for ck in self.jar:
            ck.secure = False
        try:
            with self.opener.open(req, timeout=20) as r:
                raw = r.read()
                return r.status, (json.loads(raw) if raw else None)
        except urllib.error.HTTPError as e:
            raw = e.read()
            try:
                return e.code, json.loads(raw) if raw else None
            except Exception:
                return e.code, raw.decode(errors="replace")


def check(cid, profile, flow, expect, ok, actual):
    RESULTS.append(dict(id=cid, profile=profile, flow=flow, expect=expect, actual=str(actual)[:220], ok=bool(ok)))
    print(("PASS " if ok else "FAIL ") + cid, flow, "->", str(actual)[:160])


# 1x1 JPEG: every upload in this run (listing photos, farm proof).
JPEG = bytes.fromhex("ffd8ffe000104a46494600010100000100010000ffdb004300080606070605080707070909080a0c140d0c0b0b0c1912130f141d1a1f1e1d1a1c1c20242e2720222c231c1c2837292c30313434341f27393d38323c2e333432ffc0000b080001000101011100ffc4001f0000010501010101010100000000000000000102030405060708090a0bffc400b5100002010303020403050504040000017d01020300041105122131410613516107227114328191a1082342b1c11552d1f02433627282090a161718191a25262728292a3435363738393a434445464748494a535455565758595a636465666768696a737475767778797a838485868788898a92939495969798999aa2a3a4a5a6a7a8a9aab2b3b4b5b6b7b8b9bac2c3c4c5c6c7c8c9cad2d3d4d5d6d7d8d9dae1e2e3e4e5e6e7e8e9eaf1f2f3f4f5f6f7f8f9faffda0008010100003f00fbfcffd9")
# What the browser declares when it asks for upload URLs: the photo and
# its small copy, which the URLs are then locked to.
PHOTO_SIZES = {"size": len(JPEG), "thumbSize": len(JPEG)}


def put_file(url):
    req = urllib.request.Request(url, data=JPEG, method="PUT")
    req.add_header("Content-Type", "image/jpeg")
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            return r.status
    except urllib.error.HTTPError as e:
        return e.code
    except Exception as e:
        return repr(e)


def upload_farm_proof(owner, farm_id, n=3):
    """Uploads n proof files for a farm and returns their storage keys."""
    keys = []
    for _ in range(n):
        s, b = owner.call("POST", f"/farms/{farm_id}/verification/presign", PHOTO_SIZES)
        put_file(b["uploadUrl"])
        keys.append(b["key"])
    return keys


def proof_body(keys, **over):
    """A complete submission: identity card front and back, farm photos, certified."""
    body = {"idDocumentType": "national_id", "idDocumentKey": keys[0], "idDocumentBackKey": keys[1], "photoKeys": keys[2:], "certified": True}
    body.update(over)
    return body


def open_approved_farm(owner, staff, farm):
    """Opens a farm and takes it through review: proof, then approval."""
    s, f = owner.call("POST", "/farms", farm)
    keys = upload_farm_proof(owner, f["id"])
    owner.call("POST", f"/farms/{f['id']}/verification/submit", proof_body(keys))
    staff.call("POST", f"/admin/farms/{f['id']}/review", {"decision": "approve"})
    return f


def register(role, name=None):
    time.sleep(REGISTER_PAUSE)
    c = Client()
    email = f"qa-{role}-{uuid.uuid4().hex[:8]}@betailo.test"
    pw = "Qa-" + uuid.uuid4().hex[:12]
    s, b = c.call("POST", "/auth/register", {"email": email, "password": pw, "displayName": name or ""}, token=False)
    s2, b2 = c.call("POST", "/auth/login", {"email": email, "password": pw, "remember": True}, token=False)
    c.token = (b2 or {}).get("accessToken")
    c.email, c.pw, c.user = email, pw, (b2 or {}).get("user", {})
    return c, s, b


def login(email, pw):
    c = Client()
    s, b = c.call("POST", "/auth/login", {"email": email, "password": pw, "remember": True}, token=False)
    c.token = (b or {}).get("accessToken")
    c.user = (b or {}).get("user", {})
    return c, s


now = datetime.datetime.now(datetime.timezone.utc)
iso = lambda d: d.strftime("%Y-%m-%dT%H:%M:%SZ")

# ---------------- Guest ----------------
g = Client()
s, b = g.call("GET", "/listings?pageSize=5")
check("G01", "Guest", "Browse marketplace (GET /listings)", "200 + items/total", s == 200 and isinstance(b, dict) and "items" in b, f"{s} total={b.get('total') if isinstance(b, dict) else b}")
listing_any = (b.get("items") or [{}])[0] if isinstance(b, dict) else {}
lid_any = (listing_any.get("listing") or {}).get("id")
if lid_any:
    s, b = g.call("GET", f"/listings/{lid_any}/detail")
    check("G02", "Guest", "Open listing detail", "200 with listing, cycle, farm", s == 200 and isinstance(b, dict), f"{s} keys={list(b.keys())[:8] if isinstance(b, dict) else b}")
    contact = json.dumps(b).find("contactPhone") >= 0 if isinstance(b, dict) else False
    check("G03", "Guest", "Farmer phone hidden from anonymous visitors", "no contactPhone in anonymous detail", not contact, f"contactPhone present={contact}")
s, b = g.call("GET", "/listings?q=poulets&species=jeunes-poulets")
check("G04", "Guest", "Search + species filter", "200", s == 200, f"{s} total={b.get('total') if isinstance(b, dict) else b}")
s, b = g.call("GET", "/forum/posts")
check("G05", "Guest", "Read forum", "200", s == 200, s)
for path in ["/me", "/reservations/mine", "/me/nav-counts", "/notifications", "/me/favorites"]:
    s, _ = g.call("GET", path)
    check("G06" + path, "Guest", f"Protected {path} refused", "401", s == 401, s)
s, b = g.call("POST", "/auth/register", {"email": "not-an-email", "password": "short"}, token=False)
check("G07", "Guest", "Register with invalid email/short password", "400 with message", s == 400, f"{s} {b}")
s, b = g.call("POST", "/auth/register", {"email": f"qa-esc-{uuid.uuid4().hex[:6]}@betailo.test", "password": "Qa-longpassword1", "role": "admin", "staffRole": "super_admin"}, token=False)
esc = Client()
esc.token = (g.call("POST", "/auth/login", {"email": (b or {}).get("email", ""), "password": "Qa-longpassword1"}, token=False)[1] or {}).get("accessToken")
s2, _ = esc.call("GET", "/admin/users")
check("G08", "Guest", "Self-register asking for a staff role (privilege escalation)", "account created without it, back office refused (403)", s == 201 and not (b or {}).get("staffRole") and s2 == 403, f"register={s} staffRole={(b or {}).get('staffRole')} admin={s2}")
s, b = g.call("POST", "/auth/login", {"email": "nobody-" + uuid.uuid4().hex[:6] + "@betailo.test", "password": "wrong-password"}, token=False)
check("G09", "Guest", "Login with wrong password", "401, generic message", s == 401, f"{s} {b}")

# ---------------- Farmer: onboarding + listing ----------------
adm, s = login(CREDS["ADMIN"], CREDS["PW"])
far, s, b = register("seller", "Awa Fermière")
check("F01", "Farmer", "Register a personal account", "201, not staff", s == 201 and not (b or {}).get("staffRole"), f"{s} {str(b)[:100]}")
s, me = far.call("GET", "/me")
check("F01b", "Farmer", "GET /me before any farm", "no farm, canAddFarm, farmLimit=1", s == 200 and me.get("farms") == [] and me.get("canAddFarm") is True and me.get("farmLimit") == 1, f"{s} farms={me.get('farms')} canAdd={me.get('canAddFarm')} limit={me.get('farmLimit')}")
s, b = far.call("POST", "/farms", {"name": "Ferme QA", "location": "Bafoussam", "species": ["jeunes-poulets"], "contactPhone": "677001122"})
check("F02", "Farmer", "Open a farm (POST /farms)", "201, status draft, owner can submit", s == 201 and isinstance(b, dict) and b.get("status") == "draft" and (b.get("review") or {}).get("canSubmit") is True, f"{s} {str(b)[:160]}")
farm_id = (b or {}).get("id") if isinstance(b, dict) else None
s, b = far.call("POST", "/farms", {"name": "Deuxième ferme", "location": "Douala"})
check("F02b", "Farmer", "Second farm at the default limit", "409 code farm.limit_reached", s == 409 and (b or {}).get("code") == "farm.limit_reached", f"{s} {b}")
s, me = far.call("GET", "/me")
check("F03", "Farmer", "GET /me lists the farm", "1 farm (draft), canAddFarm=false", s == 200 and len(me.get("farms", [])) == 1 and me["farms"][0].get("status") == "draft" and me.get("canAddFarm") is False, f"{s} farms={[f.get('status') for f in me.get('farms', [])]} canAdd={me.get('canAddFarm')}")
s, b = far.call("GET", f"/listings/price-hint?species=jeunes-poulets&region=Bafoussam")
check("F04", "Farmer", "Price hint", "200 or 204", s in (200, 204), f"{s} {b}")
s, b = far.call("POST", f"/farms/{farm_id}/quick-listings", {"species": "jeunes-poulets", "breed": "Brahma", "ageWeeks": 6, "quantity": 50, "unitPrice": 3000, "title": "QA Jeunes poulets", "deliveryAvailable": True})
check("F05", "Farmer", "Quick listing (sell wizard)", "201 cycle + draft listing", s == 201 and isinstance(b, dict) and b.get("listing", {}).get("status") == "draft", f"{s} status={(b or {}).get('listing', {}).get('status') if isinstance(b, dict) else b}")
qlisting = (b or {}).get("listing", {}).get("id") if isinstance(b, dict) else None
qcycle = (b or {}).get("cycle", {}).get("id") if isinstance(b, dict) else None
s, b = far.call("POST", f"/listings/{qlisting}/publish")
check("F06", "Farmer", "Publish without any photo (D2 code)", "400 code listing.photo_required", s == 400 and (b or {}).get("code") == "listing.photo_required", f"{s} {b}")
s, b = far.call("POST", f"/farms/{farm_id}/quick-listings", {"species": "jeunes-poulets", "quantity": 0, "unitPrice": -5, "title": ""})
check("F07", "Farmer", "Quick listing with quantity 0 / negative price", "400", s == 400, f"{s} {b}")
s, b = far.call("POST", f"/listings/{qlisting}/photos/presign", PHOTO_SIZES)
check("F08", "Farmer", "Presign a listing photo upload", "200 key + uploadUrl", s == 200 and isinstance(b, dict) and b.get("uploadUrl"), f"{s} {str(b)[:100]}")
key = (b or {}).get("key") if isinstance(b, dict) else None
upload = (b or {}).get("uploadUrl") if isinstance(b, dict) else None
if upload:
    # 1x1 JPEG
    jpeg = JPEG
    req = urllib.request.Request(upload, data=jpeg, method="PUT")
    req.add_header("Content-Type", "image/jpeg")
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            up = r.status
    except urllib.error.HTTPError as e:
        up = e.code
    except Exception as e:
        up = repr(e)
    check("F09", "Farmer", "Upload photo to storage via presigned URL", "2xx", isinstance(up, int) and 200 <= up < 300, up)
    tu = put_file(b["thumbUploadUrl"]) if b.get("thumbUploadUrl") else "no thumbUploadUrl"
    check("F09b", "Farmer", "Upload the photo's small copy", "2xx", isinstance(tu, int) and 200 <= tu < 300, tu)
    s2, b2 = far.call("POST", f"/listings/{qlisting}/photos/presign", {"size": 801 << 10, "thumbSize": 1000})
    check("F09c", "Farmer", "Declare a photo over 800 KB", "400 photo.too_large", s2 == 400 and (b2 or {}).get("code") == "photo.too_large", f"{s2} {b2}")
    s, b = far.call("PUT", f"/listings/{qlisting}/photos", {"keys": [key]})
    check("F10", "Farmer", "Attach photo keys to listing", "200", s == 200, f"{s} {str(b)[:100]}")
s, b = far.call("POST", f"/listings/{qlisting}/publish")
check("V01", "Farmer", "Publish from a farm staff haven't approved", "403 code farm.not_approved", s == 403 and (b or {}).get("code") == "farm.not_approved", f"{s} {b}")
s, b = far.call("POST", f"/farms/{farm_id}/verification/submit", {})
check("V02", "Farmer", "Submit the farm without proof", "400 code farm.id_document_required", s == 400 and (b or {}).get("code") == "farm.id_document_required", f"{s} {b}")
proof = upload_farm_proof(far, farm_id)
s, b = far.call("POST", f"/farms/{farm_id}/verification/submit", proof_body(proof, idDocumentKey=key))
check("V03", "Farmer", "Submit proof with a key from another upload (a listing photo)", "400", s == 400, f"{s} {b}")
s, b = far.call("POST", f"/farms/{farm_id}/verification/submit", proof_body(proof, idDocumentBackKey=""))
check("V03b", "Farmer", "Identity card without its back", "400 code farm.id_document_back_required", s == 400 and (b or {}).get("code") == "farm.id_document_back_required", f"{s} {b}")
s, b = far.call("POST", f"/farms/{farm_id}/verification/submit", proof_body(proof, certified=False))
check("V03c", "Farmer", "Submit without certifying", "400 code farm.certification_required", s == 400 and (b or {}).get("code") == "farm.certification_required", f"{s} {b}")
s, b = far.call("POST", f"/farms/{farm_id}/verification/submit", proof_body(proof))
check("V04", "Farmer", "Submit the farm for review (ID front and back + farm photo)", "200 pending with a reference", s == 200 and (b or {}).get("status") == "pending" and str((b or {}).get("reference", "")).startswith("FRM-"), f"{s} {str(b)[:140]}")
s, b = far.call("POST", f"/admin/farms/{farm_id}/review", {"decision": "approve"})
check("V05", "Farmer", "Owner approving their own farm (staff route)", "403", s == 403, f"{s} {b}")
s, b = adm.call("GET", "/admin/farms?queue=todo")
check("V06", "Admin", "Validation queue lists the pending farm with its owner", "200 includes it", s == 200 and any(f.get("id") == farm_id and f.get("owner") for f in b.get("farms", [])) and b.get("counts", {}).get("todo", 0) >= 1, f"{s} counts={b.get('counts') if isinstance(b, dict) else b}")
s, b = adm.call("GET", f"/admin/farms/{farm_id}")
check("V07", "Admin", "Farm detail carries signed links to the proof", "200 ID front + back + 1 photo", s == 200 and (b or {}).get("idDocumentUrl") and b.get("idDocumentBackUrl") and len(b.get("photoUrls", [])) == 1, f"{s} {str(b)[:120]}")
s, b = g.call("GET", f"/farms/{farm_id}")
check("V08", "Guest", "Public farm profile doesn't expose the review", "200 without review/proof", s == 200 and "review" not in b and "farm-verifications" not in json.dumps(b), f"{s} keys={sorted(b.keys()) if isinstance(b, dict) else b}")
s, b = adm.call("POST", f"/admin/farms/{farm_id}/review", {"decision": "reject"})
check("V09", "Admin", "Reject without a reason", "400 code farm.reason_required", s == 400 and (b or {}).get("code") == "farm.reason_required", f"{s} {b}")
s, b = adm.call("POST", f"/admin/farms/{farm_id}/review", {"decision": "reject", "reasonCode": "id_unreadable", "reason": "Document illisible"})
check("V10", "Admin", "Reject with a reason", "200 rejected", s == 200 and (b or {}).get("status") == "rejected", f"{s} {str(b)[:140]}")
s, me = far.call("GET", "/me")
rv = (me.get("farms") or [{}])[0].get("review") or {}
check("V11", "Farmer", "Owner sees the rejection reason and can resubmit", "code + reason + canSubmit", rv.get("rejectionReason") == "Document illisible" and rv.get("rejectionCode") == "id_unreadable" and rv.get("canSubmit") is True, f"{s} review={rv}")
check("V11b", "Farmer", "The refusal says what to redo and what was accepted", "toRedo=[id_document], accepted has photos, reviewerRole",
      rv.get("toRedo") == ["id_document"] and "photos" in (rv.get("accepted") or []) and rv.get("reviewerRole") == "super_admin", f"review={rv}")
s, b = far.call("POST", f"/farms/{farm_id}/verification/withdraw")
check("V11c", "Farmer", "Withdraw a farm that is not in the queue", "409 code farm.not_pending", s == 409 and (b or {}).get("code") == "farm.not_pending", f"{s} {b}")
redo = upload_farm_proof(far, farm_id, 2)
s, b = far.call("POST", f"/farms/{farm_id}/verification/submit", {"idDocumentType": "national_id", "idDocumentKey": redo[0], "idDocumentBackKey": redo[1], "certified": True})
check("V12", "Farmer", "Resubmit only the refused document, the accepted photos are kept", "200 pending", s == 200 and (b or {}).get("status") == "pending", f"{s} {str(b)[:120]}")
s, b = far.call("POST", f"/farms/{farm_id}/verification/withdraw")
check("V12c", "Farmer", "Withdraw the request", "200 draft, can submit", s == 200 and (b or {}).get("status") == "draft" and (b.get("review") or {}).get("canSubmit") is True, f"{s} {str(b)[:120]}")
s, b = far.call("POST", f"/farms/{farm_id}/verification/submit", proof_body(proof))
check("V12d", "Farmer", "Send it again after withdrawing", "200 pending", s == 200 and (b or {}).get("status") == "pending", f"{s} {str(b)[:120]}")
s, b = adm.call("GET", "/admin/farms?queue=resubmitted")
check("V12b", "Admin", "The resubmitted farm is in its own queue", "200 includes it", s == 200 and any(f.get("id") == farm_id for f in b.get("farms", [])), f"{s} counts={b.get('counts') if isinstance(b, dict) else b}")
s, b = adm.call("POST", f"/admin/farms/{farm_id}/review", {"decision": "approve", "checks": ["name_matches", "photos_genuine", "not-a-check"]})
check("V13", "Admin", "Approve the farm", "200 approved", s == 200 and (b or {}).get("status") == "approved", f"{s} {str(b)[:120]}")
s, b = adm.call("GET", f"/admin/farms/{farm_id}")
check("V13b", "Admin", "The ticked review points are kept with the decision", "the two known checks, 4 points listed",
      s == 200 and b.get("checks") == ["name_matches", "photos_genuine"] and len(b.get("reviewChecks", [])) == 4, f"{s} checks={(b or {}).get('checks')}")
s, b = adm.call("GET", "/admin/farms?queue=done")
check("V13c", "Admin", "Queue rows carry a thumbnail of the farm", "thumbUrl on the reviewed farm",
      s == 200 and any(f.get("id") == farm_id and f.get("thumbUrl") for f in b.get("farms", [])), f"{s}")
s, b = far.call("GET", "/notifications")
check("V14", "Farmer", "Owner is notified of the decisions", "farm_rejected and farm_approved in the inbox", s == 200 and {"farm_rejected", "farm_approved"} <= {n.get("type") for n in b.get("items", [])}, f"{s} types={[n.get('type') for n in b.get('items', [])]}")
s, b = far.call("POST", f"/listings/{qlisting}/publish")
check("F11", "Farmer", "Publish listing with a photo", "200 published", s == 200 and isinstance(b, dict) and b.get("status") == "published", f"{s} {(b or {}).get('status') if isinstance(b, dict) else b}")
s, b = far.call("GET", f"/farms/{farm_id}/listings")
check("F12", "Farmer", "Owner listings view (all statuses)", "200 with the new listing", s == 200 and any(r["listing"]["id"] == qlisting for r in (b or [])), f"{s} n={len(b) if isinstance(b, list) else b}")
s, b = far.call("POST", f"/cycles/{qcycle}/health-events", {"title": "Vaccin Newcastle", "date": iso(now)})
check("F13", "Farmer", "Add health record", "201", s in (200, 201), f"{s}")
s, b = far.call("GET", f"/cycles/{qcycle}")
check("F14", "Farmer", "Cycle detail with timeline + nextStatuses", "timeline has the record; nextStatuses non-empty", s == 200 and any(t.get("title") == "Vaccin Newcastle" for t in b.get("timeline", [])) and b.get("nextStatuses"), f"{s} next={b.get('nextStatuses') if isinstance(b, dict) else b}")
s, b = far.call("POST", f"/cycles/{qcycle}/health-events", {"title": "Futur", "date": iso(now + datetime.timedelta(days=30))})
check("F15", "Farmer", "Planned health record (30 days ahead) accepted, far future refused", "201 then 400",
      s == 201 and far.call("POST", f"/cycles/{qcycle}/health-events", {"title": "Trop loin", "date": iso(now + datetime.timedelta(days=90))})[0] == 400, f"{s}")

# ---------------- Buyer ----------------
buy, s, b = register("buyer", "Jean Acheteur")
check("B01", "Buyer", "Register + login", "201 / token", s == 201 and buy.token, s)
s, b = buy.call("PATCH", "/me", {"displayName": "Jean A.", "phone": "690112233", "city": "Douala"})
check("B02", "Buyer", "Edit profile", "200, phone normalized", s == 200 and isinstance(b, dict) and str(b.get("phone", "")).startswith("+237"), f"{s} phone={b.get('phone') if isinstance(b, dict) else b}")
s, b = buy.call("PATCH", "/me", {"displayName": "x" * 300, "phone": "12", "city": ""})
check("B03", "Buyer", "Edit profile with invalid phone / overlong name", "400", s == 400, f"{s} {b}")
s, b = buy.call("PUT", f"/listings/{qlisting}/favorite")
check("B04", "Buyer", "Add favorite", "2xx", 200 <= s < 300, s)
s, b = buy.call("GET", "/me/favorites")
check("B05", "Buyer", "List favorites", "contains listing", s == 200 and bool(qlisting) and qlisting in json.dumps(b), s)
s, b = buy.call("POST", f"/listings/{qlisting}/quote", {"quantity": 10, "payNow": True})
check("B06", "Buyer", "Price quote", "200 with total incl. fee", s == 200 and isinstance(b, dict), f"{s} {str(b)[:140]}")
s, b = buy.call("POST", f"/listings/{qlisting}/reservations", {"quantity": 999})
check("B07", "Buyer", "Waiting-list request larger than the lot", "400 reservation.above_capacity", s == 400 and (b or {}).get("code") == "reservation.above_capacity", f"{s} {b}")
s, b = buy.call("POST", f"/listings/{qlisting}/reservations", {"quantity": 5})
check("B08", "Buyer", "Reserve, pay later", "201 pending", s == 201 and b["reservation"]["status"] == "pending", f"{s} {(b or {}).get('reservation', {}).get('status') if isinstance(b, dict) else b}")
r_unpaid = b["reservation"]["id"] if s == 201 else None
s, d = buy.call("GET", f"/listings/{qlisting}/detail")
check("W01", "Buyer", "Unpaid reservation holds no places", "available unchanged (50)", s == 200 and d["cycle"]["availableQuantity"] == 50, f"{s} available={d.get('cycle', {}).get('availableQuantity') if isinstance(d, dict) else d}")
s, b = buy.call("GET", "/me/nav-counts")
check("B09", "Buyer", "Nav badge counts the unpaid reservation", "buyer=1", s == 200 and b.get("buyer") == 1, f"{s} {b}")
s, b = buy.call("POST", f"/reservations/{r_unpaid}/pay", {"paymentMethod": "mtn_momo", "payerPhone": "690112233"})
check("B10", "Buyer", "Pay the pending reservation (MoMo sandbox)", "200, payment held", s == 200, f"{s} {str(b)[:140]}")
s, b = buy.call("POST", f"/listings/{qlisting}/reservations", {"quantity": 3})
r_cancel = b["reservation"]["id"] if s == 201 else None
s, b = buy.call("POST", f"/reservations/{r_cancel}/cancel")
check("B11", "Buyer", "Cancel an unpaid reservation", "200", s == 200, f"{s} {str(b)[:100]}")
s, b = buy.call("POST", f"/listings/{qlisting}/reservations", {"quantity": 2, "payNow": True, "paymentMethod": "orange_money", "payerPhone": "655001122"})
check("B12", "Buyer", "Reserve and pay now (Orange Money)", "201 confirmed + held", s == 201 and b["reservation"]["status"] == "confirmed", f"{s} {(b or {}).get('reservation', {}).get('status') if isinstance(b, dict) else b}")
r_paid2 = b["reservation"]["id"] if s == 201 else None
s, b = buy.call("POST", f"/reservations/{r_paid2}/reviews", {"rating": 5, "criteria": {"health": 5, "conformity": 5, "communication": 5, "timeliness": 5}})
check("B13", "Buyer", "Review before delivery", "refused", s >= 400, f"{s} {b}")
s, b = buy.call("POST", f"/reservations/{r_paid2}/confirm-delivery")
check("B14", "Buyer", "Confirm delivery before farmer marks delivered", "refused", s >= 400, f"{s} {b}")
s, b = buy.call("GET", "/reservations/mine")
check("B15", "Buyer", "My reservations with server actions", "200 with actions per item", s == 200 and all("actions" in i for i in b.get("items", [])), f"{s} counts={b.get('counts') if isinstance(b, dict) else b}")
s, b = buy.call("PUT", f"/listings/{qlisting}", {"title": "hacked", "unitPrice": 1})
check("B16", "Buyer", "Edit someone else's listing (IDOR)", "403", s in (403, 404), f"{s} {b}")
s, b = buy.call("GET", f"/farms/{farm_id}/reservations")
check("B17", "Buyer", "Read a farm's reservations (IDOR)", "403", s in (403, 404), f"{s}")
s, b = buy.call("GET", "/admin/overview")
check("B18", "Buyer", "Open admin API", "403", s == 403, s)
s, b = buy.call("POST", "/forum/posts", {"category": "sante", "title": "Vaccin pour poussins ?", "body": "Quel calendrier conseillez-vous ?"})
check("B19", "Buyer", "Create forum post", "201", s == 201, f"{s} {str(b)[:80]}")
post_id = (b or {}).get("id") if isinstance(b, dict) else None
s, b = buy.call("POST", "/reports", {"targetId": qlisting, "targetType": "listing", "reason": "Photos ne correspondent pas"})
check("B20", "Buyer", "Report a listing", "2xx", 200 <= s < 300, f"{s} {str(b)[:80]}")
report_id = (b or {}).get("id") if isinstance(b, dict) else None
s, b = buy.call("GET", "/notifications")
check("B21", "Buyer", "Notifications inbox", "200", s == 200, f"{s} n={len(b.get('items', [])) if isinstance(b, dict) else b}")

# ---------------- Farmer: fulfilment ----------------
s, b = far.call("POST", f"/listings/{qlisting}/reservations", {"quantity": 1})
check("F16", "Farmer", "Reserve own listing", "refused", s >= 400, f"{s} {b}")
s, b = far.call("GET", "/me/nav-counts")
check("F17", "Farmer", "Seller badge counts paid orders to deliver", "seller=2", s == 200 and b.get("seller") == 2, f"{s} {b}")
s, b = far.call("GET", f"/farms/{farm_id}/dashboard")
check("F18", "Farmer", "Dashboard with À faire", "todos contains mark_delivered", s == 200 and any(t["kind"] == "mark_delivered" for t in b.get("todos", [])), f"{s} todos={b.get('todos') if isinstance(b, dict) else b}")
s, b = far.call("PUT", f"/cycles/{qcycle}", {"startsAt": iso(now), "capacity": 3, "minPerReservation": 0, "maxPerReservation": 0})
check("F19", "Farmer", "Lower capacity below what is reserved", "refused", s >= 400, f"{s} {b}")
s, b = far.call("POST", f"/reservations/{r_unpaid}/mark-delivered")
check("F20", "Farmer", "Mark order delivered", "200", s == 200, f"{s} {str(b)[:100]}")
s, b = far.call("PATCH", f"/cycles/{qcycle}/status", {"status": "planned"})
check("F21", "Farmer", "Invalid cycle status transition", "400", s == 400, f"{s} {b}")

# ---------------- Buyer: close the loop ----------------
s, b = buy.call("GET", "/me/nav-counts")
check("B22", "Buyer", "Badge counts the delivery to confirm", "buyer=1", s == 200 and b.get("buyer") == 1, f"{s} {b}")
s, b = buy.call("POST", f"/reservations/{r_unpaid}/confirm-delivery")
check("B23", "Buyer", "Confirm delivery (escrow release)", "200, released", s == 200, f"{s} {str(b)[:120]}")
s, b = buy.call("POST", f"/reservations/{r_unpaid}/reviews", {"rating": 4, "comment": "Bons poulets, livrés à l'heure", "criteria": {"health": 4, "conformity": 5, "communication": 4, "timeliness": 5}, "tags": []})
check("B24", "Buyer", "Leave a review after release", "201", s == 201, f"{s} {str(b)[:120]}")
review_id = (b or {}).get("id") if isinstance(b, dict) else None
s, b = buy.call("POST", f"/reservations/{r_unpaid}/reviews", {"rating": 1, "criteria": {"health": 1, "conformity": 1, "communication": 1, "timeliness": 1}})
check("B25", "Buyer", "Second review on the same reservation", "409 review.already_reviewed", s == 409 and (b or {}).get("code") == "review.already_reviewed", f"{s} {b}")
s, b = far.call("POST", f"/reviews/{review_id}/reply", {"text": "Merci pour votre confiance"})
check("F22", "Farmer", "Reply to a review", "200", s in (200, 201), f"{s} {str(b)[:100]}")
s, b = buy.call("POST", f"/reservations/{r_paid2}/dispute", {"reason": "Animaux malades"})
check("B26", "Buyer", "Open a dispute on a paid order", "2xx", 200 <= s < 300, f"{s} {str(b)[:100]}")

# ---------------- Admin ----------------
adm, s = login(CREDS["ADMIN"], CREDS["PW"])
check("A01", "Admin", "Super admin login", "200 staffRole=super_admin", s == 200 and adm.user.get("staffRole") == "super_admin", f"{s} staffRole={adm.user.get('staffRole')}")
for path in ["/admin/overview", "/admin/overview?period=7d", "/admin/overview?period=12m", "/admin/nav-counts", "/admin/users", "/admin/listings", "/admin/escrow", "/admin/reports", "/admin/cycles", "/admin/farms?queue=done", "/admin/staff", "/admin/farm-limits", "/admin/audit"]:
    s, b = adm.call("GET", path)
    check("A02" + path, "Admin", f"GET {path}", "200", s == 200, s)
s, b = far.call("GET", "/farms/mine/cards")
cards = b if isinstance(b, list) else []
check("F30", "Farmer", "My farms with their figures", "200, stats on each card",
      s == 200 and len(cards) > 0 and all("activeCycles" in c and "salesCount" in c and "trustScore" in c and "review" in c for c in cards), f"{s} n={len(cards)}")
s, b = far.call("GET", "/me/seller-dashboard")
check("F32", "Farmer", "Combined dashboard over every farm", "200 with farmCount and the new figures",
      s == 200 and isinstance(b, dict) and b.get("farmCount", 0) >= 1 and all(k in b for k in ("reservationsPreviousPeriod", "heldReservations", "newReservations", "reviewCount")), f"{s} farms={(b or {}).get('farmCount')}")
s, b = far.call("GET", "/me/nav-counts")
check("F33", "Farmer", "Nav counts carry active cycles", "200 cycles", s == 200 and "cycles" in (b or {}), f"{s} {b}")
s, b = far.call("GET", f"/farms/{farm_id}/reservations?q=zzz-no-such-reference")
check("F34", "Farmer", "Order search narrows the tab, counts stay whole", "200, no item", s == 200 and b.get("items") == [] and "counts" in b, f"{s} n={len((b or {}).get('items', []))}")
s, b = g.call("GET", "/me/seller-dashboard", token=False)
check("F35", "Guest", "Combined dashboard needs a session", "401", s == 401, s)
s, b = g.call("GET", "/farms/mine/cards", token=False)
check("F31", "Guest", "My farms needs a session", "401", s == 401, s)
if report_id:
    s, b = adm.call("GET", f"/admin/reports/{report_id}")
    check("A03", "Admin", "Report detail with risk signals", "200 riskScore", s == 200 and "riskScore" in json.dumps(b), f"{s}")
    row = (b or {}).get("report", {})
    check("A03b", "Admin", "Report detail names its target, level and listing", "targetLabel, level, listing, subject",
          bool(row.get("targetLabel")) and row.get("level") in ("high", "medium", "low") and bool((b or {}).get("listing")) and bool((b or {}).get("subject")),
          f"label={row.get('targetLabel')} level={row.get('level')} listing={bool((b or {}).get('listing'))} subject={bool((b or {}).get('subject'))}")
    s, b = adm.call("GET", "/admin/reports")
    rows = b if isinstance(b, list) else []
    check("A03c", "Admin", "Report queue rows carry count, level and reporter", "every row",
          s == 200 and len(rows) > 0 and all(r.get("count", 0) >= 1 and r.get("level") and r.get("createdAt") for r in rows), f"{s} n={len(rows)}")
    s, b = adm.call("POST", f"/admin/reports/{report_id}/resolve", {"action": "dismiss"})
    check("A04", "Admin", "Resolve report (dismiss)", "200", s == 200, f"{s} {str(b)[:100]}")
s, b = adm.call("POST", f"/admin/users/{far.user.get('id')}/farm-limit", {"limit": 2})
check("A05a", "Admin", "Farm limit exception without a reason", "400 code admin.farm_limit_reason_required", s == 400 and (b or {}).get("code") == "admin.farm_limit_reason_required", f"{s} {b}")
s, b = adm.call("POST", f"/admin/users/{far.user.get('id')}/farm-limit", {"limit": 2, "reason": "Second site"})
check("A05", "Admin", "Raise one account's farm limit, with a reason", "200 farmLimit=2", s == 200 and (b or {}).get("farmLimit") == 2 and (b.get("farmLimitNote") or {}).get("reason") == "Second site", f"{s} {str(b)[:100]}")
s, b = far.call("POST", "/farms", {"name": "Deuxième ferme", "location": "Douala"})
check("A05b", "Farmer", "Second farm once staff raised the limit", "201 draft", s == 201 and (b or {}).get("status") == "draft", f"{s} {str(b)[:100]}")
s, b = far.call("POST", "/farms", {"name": "Troisième ferme", "location": "Douala"})
check("A05c", "Farmer", "Third farm beyond the raised limit", "409 code farm.limit_reached", s == 409 and (b or {}).get("code") == "farm.limit_reached", f"{s} {b}")
s, b = adm.call("POST", "/farms", {"name": "Ferme du staff", "location": "Douala"})
check("A05d", "Admin", "Staff account opening a farm", "403 code farm.staff_cannot_sell", s == 403 and (b or {}).get("code") == "farm.staff_cannot_sell", f"{s} {b}")
s, b = adm.call("POST", f"/admin/users/{far.user.get('id')}/farm-limit", {"limit": 1, "reason": "Erreur"})
check("A05e", "Admin", "Limit below the farms the account owns", "409 code admin.farm_limit_below_owned", s == 409 and (b or {}).get("code") == "admin.farm_limit_below_owned", f"{s} {b}")
s, b = adm.call("GET", "/admin/farm-limits")
check("A05f", "Admin", "Exceptions list names who granted it", "the account, with setByRole", s == 200 and any(e.get("id") == far.user.get("id") and (e.get("farmLimitNote") or {}).get("setByRole") == "super_admin" for e in b.get("exceptions", [])), f"{s} default={b.get('default') if isinstance(b, dict) else b}")
s, b = adm.call("GET", "/admin/escrow")
held = [e for e in (b.get("items", b) if isinstance(b, dict) else b or []) if isinstance(e, dict) and r_paid2 and r_paid2 in json.dumps(e)]
s, b = adm.call("POST", f"/admin/escrow/{r_paid2}/refund")
check("A06", "Admin", "Refund a disputed escrow", "200", s == 200, f"{s} {str(b)[:140]}")
victim, _, _ = register("buyer")
s, b = adm.call("POST", f"/admin/users/{victim.user.get('id')}/suspend")
check("A07", "Admin", "Suspend a user", "200", s == 200, f"{s} {str(b)[:100]}")
_, s = login(victim.email, victim.pw)
check("A08", "Admin", "Suspended user cannot log in", "401/403", s in (401, 403), s)
s, b = victim.call("GET", "/me")
check("A09", "Admin", "Suspended user's existing session stops working (D1)", "401 account_suspended", s == 401 and (b or {}).get("code") == "account_suspended", f"{s} {b}")
s, b = adm.call("POST", f"/admin/users/{victim.user.get('id')}/reactivate")
check("A10", "Admin", "Lift the suspension", "200 suspended=false", s == 200 and (b or {}).get("suspended") is False, f"{s} {str(b)[:100]}")
_, s = login(victim.email, victim.pw)
check("A11", "Admin", "Reactivated user logs in again", "200", s == 200, s)

# ---------------- Staff roles ----------------
s, me = adm.call("GET", "/me")
check("T01", "Staff", "The startup account is the root super admin", "isRoot, every permission", s == 200 and me.get("isRoot") is True and "staff.manage" in me.get("permissions", []), f"{s} isRoot={me.get('isRoot')} n={len(me.get('permissions', []))}")
s, b = adm.call("POST", f"/admin/staff/{me['id']}/role", {"role": "support"})
check("T02", "Staff", "Nobody changes the root account's role", "403 code admin.root_protected", s == 403 and (b or {}).get("code") == "admin.root_protected", f"{s} {b}")
s, b = adm.call("POST", f"/admin/users/{me['id']}/suspend")
check("T03", "Staff", "Nobody suspends the root account", "403 code admin.root_protected", s == 403 and (b or {}).get("code") == "admin.root_protected", f"{s} {b}")
mod, _, _ = register("buyer", "Rita Modération")
s, b = mod.call("GET", "/admin/overview")
check("T04", "Staff", "An ordinary account in the back office", "403", s == 403, f"{s}")
s, b = adm.call("POST", "/admin/staff", {"email": mod.email, "role": "moderator"})
check("T05", "Staff", "Grant the moderator role to an existing account", "200 staffRole=moderator", s == 200 and (b or {}).get("staffRole") == "moderator", f"{s} {str(b)[:100]}")
s, b = mod.call("GET", "/admin/farms?queue=todo")
check("T06", "Staff", "The role applies to the session already open", "200", s == 200, f"{s}")
s, b = mod.call("GET", "/admin/escrow")
check("T07", "Staff", "Moderator reading escrow", "403 admin.permission_denied naming finance", s == 403 and (b or {}).get("code") == "admin.permission_denied" and "finance" in (b.get("params") or {}).get("roles", []), f"{s} {b}")
s, b = mod.call("GET", "/admin/escrow/export.csv")
check("T07b", "Staff", "Moderator downloading the accounting export", "403 admin.permission_denied", s == 403 and (b or {}).get("code") == "admin.permission_denied", f"{s} {b}")
s, b = mod.call("POST", "/admin/staff", {"email": victim.email, "role": "support"})
check("T08", "Staff", "Moderator granting a role", "403", s == 403, f"{s} {b}")
s, b = mod.call("POST", "/farms", {"name": "Ferme du modérateur", "location": "Douala"})
check("T09", "Staff", "Moderator opening a farm", "403 code farm.staff_cannot_sell", s == 403 and (b or {}).get("code") == "farm.staff_cannot_sell", f"{s} {b}")
s, b = adm.call("POST", "/admin/staff", {"email": far.email, "role": "support"})
check("T10", "Staff", "A farm owner joining the staff", "409 code admin.staff_owns_farm", s == 409 and (b or {}).get("code") == "admin.staff_owns_farm", f"{s} {b}")
s, b = adm.call("POST", f"/admin/staff/{mod.user.get('id')}/role", {"role": "finance"})
check("T11", "Staff", "Change a member's role", "200 staffRole=finance", s == 200 and (b or {}).get("staffRole") == "finance", f"{s} {str(b)[:100]}")
s, b = mod.call("GET", "/admin/escrow")
s2, b2 = mod.call("GET", "/admin/reports")
check("T12", "Staff", "Finance reads escrow, not reports", "200 then 403", s == 200 and s2 == 403, f"{s} {s2}")
req = urllib.request.Request(BASE + "/admin/escrow/export.csv", headers={"Authorization": "Bearer " + mod.token})
try:
    with urllib.request.urlopen(req, timeout=20) as r:
        csv_status, csv_type, csv_body = r.status, r.headers.get("Content-Type", ""), r.read().decode("utf-8-sig")
except urllib.error.HTTPError as e:
    csv_status, csv_type, csv_body = e.code, "", ""
check("T12b", "Staff", "Finance downloads the accounting export", "200 text/csv with a header row and rows",
      csv_status == 200 and csv_type.startswith("text/csv") and csv_body.startswith("reservation,buyer,farm,amount_fcfa") and csv_body.count("\n") >= 2, f"{csv_status} {csv_type} lines={csv_body.count(chr(10))}")
s, b = adm.call("GET", "/admin/staff")
root_row = next((m for m in b.get("members", []) if m.get("isRoot")), {})
check("T13b", "Staff", "Team page says when a member last acted", "lastActiveAt on the root", bool(root_row.get("lastActiveAt")), f"{root_row.get('lastActiveAt')}")
check("T13", "Staff", "Team page: members and the role table", "root + the new member, 4 roles", s == 200 and len(b.get("members", [])) >= 2 and len(b.get("roles", [])) == 4, f"{s} members={len(b.get('members', [])) if isinstance(b, dict) else b}")
s, b = adm.call("POST", f"/admin/staff/{mod.user.get('id')}/role", {"role": ""})
check("T14", "Staff", "Revoke a role", "200 no staffRole", s == 200 and not (b or {}).get("staffRole"), f"{s} {str(b)[:100]}")
s, b = mod.call("GET", "/admin/overview")
check("T15", "Staff", "A revoked member is locked out at once", "403", s == 403, f"{s}")
s, b = adm.call("GET", "/admin/audit")
actions = {a.get("action") for a in (b if isinstance(b, list) else [])}
wanted = {"farm.approve", "farm.reject", "farm.proof_viewed", "user.suspend", "user.reactivate", "user.farm_limit", "escrow.refund", "staff.grant", "staff.revoke"}
check("T16", "Staff", "The journal recorded what staff did in this run", "every action above", s == 200 and wanted <= actions, f"{s} missing={sorted(wanted - actions)}")
s, b = adm.call("POST", f"/listings/{qlisting}/reservations", {"quantity": 1})
check("A10", "Admin", "Admin places a reservation", "documented behavior (allowed or refused)", True, f"{s} {str(b)[:100]}")


# ---------------- Regression of QA defects + new rules ----------------
s, b = g.call("GET", "/reference/geo")
check("R01", "Guest", "Geo reference lists Cameroon's regions and calling codes", "200, CM with 10 regions, GH calling code 233", s == 200 and any(c["code"] == "CM" and len(c["regions"]) == 10 for c in b.get("countries", [])) and (b.get("callingCodes") or {}).get("GH") == "233", f"{s}")
s, b = g.call("GET", "/listings?pageSize=5")
check("R02", "Guest", "Search returns facets", "species/cycleTypes/regions/price facets", s == 200 and all(k in (b.get("facets") or {}) for k in ("species", "cycleTypes", "statuses", "regions", "priceMin", "priceMax")), f"{s} {str(b.get('facets'))[:160]}")
s, b = g.call("GET", "/listings?species=jeunes-poulets,poussins&status=planned,in_progress&available=" + now.strftime("%Y-%m"))
check("R03", "Guest", "Multi-value filters + availability month", "200", s == 200 and isinstance(b.get("items"), list), f"{s} total={b.get('total')}")
s, b = g.call("GET", "/listings?available=octobre")
check("R04", "Guest", "Invalid availability month", "400 search.available_invalid", s == 400 and (b or {}).get("code") == "search.available_invalid", f"{s} {b}")
s, b = g.call("GET", "/listings?q=QA%20Bafoussam")
check("R05", "Guest", "Word search across title and town", "finds the QA listing", s == 200 and any(i["listing"]["id"] == qlisting for i in b.get("items", [])), f"{s} total={b.get('total')}")
s, b = far.call("PUT", f"/farms/{farm_id}", {"name": "Ferme QA", "location": "Bafoussam", "region": "Atlantique"})
check("R07", "Farmer", "Farm with an unknown Cameroonian region", "400 farm.region_invalid", s == 400 and (b or {}).get("code") == "farm.region_invalid", f"{s} {b}")
s, b = far.call("PUT", f"/farms/{farm_id}", {"name": "Ferme QA", "location": "Bafoussam", "country": "CM", "region": "ouest"})
check("R08", "Farmer", "Farm region is stored in canonical form", "Ouest", s == 200 and b.get("region") == "Ouest" and b.get("country") == "CM", f"{s} {b.get('region') if isinstance(b, dict) else b}")

s, b = g.call("GET", "/listings?country=CM&region=Ouest")
check("R06", "Guest", "Region filter (farm in Bafoussam is Ouest)", "includes the QA listing", s == 200 and all(i.get("farmRegion") == "Ouest" for i in b.get("items", [])) and any(i["listing"]["id"] == qlisting for i in b.get("items", [])), f"{s} total={b.get('total')}")

# Waiting list: a full lot, an unpaid buyer, a paying buyer takes the places.
wl_farmer, _, _ = register("seller", "Waitlist Farmer")
wf = open_approved_farm(wl_farmer, adm, {"name": "Ferme WL", "location": "Dschang", "region": "Ouest"})
s, ql = wl_farmer.call("POST", f"/farms/{wf['id']}/quick-listings", {"species": "poussins", "quantity": 10, "unitPrice": 700, "title": "Lot liste d'attente", "deliveryAvailable": False})
wl_listing, wl_cycle = ql["listing"]["id"], ql["cycle"]["id"]
s, up = wl_farmer.call("POST", f"/listings/{wl_listing}/photos/presign", PHOTO_SIZES)
req = urllib.request.Request(up["uploadUrl"], data=jpeg, method="PUT"); req.add_header("Content-Type", "image/jpeg"); urllib.request.urlopen(req, timeout=20)
wl_farmer.call("PUT", f"/listings/{wl_listing}/photos", {"keys": [up["key"]]})
wl_farmer.call("POST", f"/listings/{wl_listing}/publish")
waiter, _, _ = register("buyer", "Waiting Buyer")
payer, _, _ = register("buyer", "Paying Buyer")
s, w = waiter.call("POST", f"/listings/{wl_listing}/reservations", {"quantity": 6})
check("W02", "Buyer", "Join the waiting list", "201 pending", s == 201 and w["reservation"]["status"] == "pending", f"{s}")
wid = w["reservation"]["id"] if s == 201 else None
s, d = g.call("GET", f"/listings/{wl_listing}/detail")
check("W03", "Buyer", "Waiting list takes no places", "10 available", d["cycle"]["availableQuantity"] == 10, f"available={d['cycle']['availableQuantity']}")
s, p = payer.call("POST", f"/listings/{wl_listing}/reservations", {"quantity": 10, "payNow": True, "paymentMethod": "mtn_momo", "payerPhone": "677112233"})
check("W04", "Buyer", "A paying buyer takes every place", "201 confirmed", s == 201 and p["reservation"]["status"] == "confirmed", f"{s}")
s, b = waiter.call("POST", f"/reservations/{wid}/pay", {"paymentMethod": "mtn_momo", "payerPhone": "690112233"})
check("W05", "Buyer", "Waiting buyer pays after the lot sold out", "409 reservation.sold_out", s == 409 and (b or {}).get("code") == "reservation.sold_out", f"{s} {b}")
s, b = wl_farmer.call("PATCH", f"/cycles/{wl_cycle}/status", {"status": "done"})
check("W06", "Farmer", "Close the lot", "200 done", s == 200 and b.get("status") == "done", f"{s}")
s, mine = waiter.call("GET", "/reservations/mine?group=cancelled")
check("W07", "Buyer", "Closing the lot cancels the waiting list", "waiting reservation cancelled", s == 200 and bool(wid) and wid in json.dumps(mine), f"{s} cancelled={len(mine.get('items', []))}")
s, n = waiter.call("GET", "/notifications")
check("W08", "Buyer", "Waiting buyer is notified", "reservation_expired notification", s == 200 and any(x.get("type") == "reservation_expired" for x in n.get("items", [])), f"{s}")
s, b = g.call("GET", f"/listings?species=poussins&q=attente")
check("W09", "Guest", "A closed lot leaves the marketplace", "not listed", s == 200 and not any(i["listing"]["id"] == wl_listing for i in b.get("items", [])), f"{s} total={b.get('total')}")
s, b = waiter.call("POST", f"/listings/{wl_listing}/quote", {"quantity": 2, "payNow": True})
check("R09", "Buyer", "Quote on a closed lot", "409 reservation.cycle_closed", s == 409 and (b or {}).get("code") == "reservation.cycle_closed", f"{s} {b}")
s, b = buy.call("POST", f"/listings/{qlisting}/reservations", {"quantity": 1, "payNow": True, "paymentMethod": "mtn_momo", "payerPhone": "12"})
check("R10", "Buyer", "Invalid Mobile Money number", "400 payment.phone_invalid", s == 400 and (b or {}).get("code") == "payment.phone_invalid", f"{s} {b}")

# ---------------- Account security ----------------
s, b = buy.call("POST", "/me/password", {"currentPassword": "wrong", "newPassword": "Qa-newpassword-1", "remember": True})
check("S01", "Buyer", "Change password with wrong current password", "401/400", s in (400, 401, 403), f"{s} {b}")
old_token = buy.token
s, b = buy.call("POST", "/me/password", {"currentPassword": buy.pw, "newPassword": "Qa-newpassword-1", "remember": True})
check("S02", "Buyer", "Change password", "200 new session", s == 200 and isinstance(b, dict) and b.get("accessToken"), f"{s}")
_, s = login(buy.email, buy.pw)
check("S03", "Buyer", "Old password no longer works", "401", s == 401, s)
s, b = g.call("GET", "/listings?pageSize=1000")
check("S04", "Guest", "Oversized pageSize is capped", "200, ≤100 items", s == 200 and len(b.get("items", [])) <= 100, f"{s} n={len(b.get('items', [])) if isinstance(b, dict) else b}")

if os.environ.get("QA_RESULTS"):
    json.dump(RESULTS, open(os.environ["QA_RESULTS"], "w"), indent=1, ensure_ascii=False)
# ---------------- Moderation decisions ----------------
rep1, _, _ = register("buyer", "Serge Signale")
s, b = rep1.call("POST", "/reports", {"targetId": qlisting, "targetType": "listing", "reason": "Paiement hors plateforme demandé"})
mod_report = (b or {}).get("id") if isinstance(b, dict) else None
s, d = adm.call("GET", f"/admin/reports/{mod_report}")
row = (d or {}).get("report", {})
check("M01", "Admin", "A report has a reference and the decisions it can take", "RPT- reference; dismiss, hide_listing, reverify, suspend",
      s == 200 and str(row.get("reference", "")).startswith("RPT-") and {"dismiss", "hide_listing", "reverify", "suspend"} <= set(d.get("actions", [])), f"{s} ref={row.get('reference')} actions={d.get('actions')}")
s, b = adm.call("PUT", f"/admin/reports/{mod_report}/note", {"note": "Photos identiques à une autre annonce."})
check("M02", "Admin", "Keep an internal note on a report", "200 note saved", s == 200 and (b or {}).get("note") == "Photos identiques à une autre annonce.", f"{s} {str(b)[:120]}")
s, b = adm.call("PUT", f"/admin/reports/{mod_report}/note", {"note": "x" * 1001})
check("M03", "Admin", "A note longer than the limit", "400 code admin.report_note_too_long", s == 400 and (b or {}).get("code") == "admin.report_note_too_long", f"{s} {b}")
s, q = adm.call("GET", "/admin/reports")
rows = [r for r in (q if isinstance(q, list) else []) if r.get("targetId") == qlisting]
check("M04", "Admin", "The queue has one row per target", "one row for the listing, counting its reports", len(rows) == 1 and rows[0].get("count", 0) >= 2, f"rows={len(rows)} count={rows[0].get('count') if rows else None}")
s, b = adm.call("POST", f"/admin/reports/{mod_report}/resolve", {"action": "hide_listing"})
check("M05", "Admin", "Hide the reported listing", "200 handled", s == 200 and (b or {}).get("status") == "handled", f"{s} {str(b)[:120]}")
s, b = g.call("GET", f"/listings/{qlisting}", token=False)
check("M06", "Guest", "A hidden listing is off the market", "unpublished or not found", (s == 200 and (b or {}).get("status") == "unpublished") or s == 404, f"{s} {(b or {}).get('status') if isinstance(b, dict) else b}")
s, d = adm.call("GET", f"/admin/reports/{mod_report}")
check("M07", "Admin", "A decided report takes no further decision", "no action", s == 200 and d.get("actions") == [], f"{s} actions={d.get('actions')}")
rep2, _, _ = register("buyer", "Nadège Signale")
s, b = rep2.call("POST", "/reports", {"targetId": qlisting, "targetType": "listing", "reason": "Identité douteuse"})
again = (b or {}).get("id") if isinstance(b, dict) else None
s, b = adm.call("POST", f"/admin/reports/{again}/resolve", {"action": "reverify", "note": "Vérifier la pièce."})
check("M08", "Admin", "Ask the farm's owner to verify their identity again", "200 handled", s == 200 and (b or {}).get("status") == "handled", f"{s} {str(b)[:120]}")
s, me = far.call("GET", "/me")
asked = next((f for f in me.get("farms", []) if f.get("id") == farm_id), {})
check("M09", "Farmer", "The owner is asked for a new identity document", "rejected, code reverification, id_document to redo",
      asked.get("status") == "rejected" and (asked.get("review") or {}).get("rejectionCode") == "reverification" and (asked.get("review") or {}).get("toRedo") == ["id_document"], f"status={asked.get('status')} review={asked.get('review')}")
s, b = rep1.call("PUT", f"/admin/reports/{mod_report}/note", {"note": "x"})
check("M10", "Buyer", "An ordinary account writing a staff note", "403", s == 403, f"{s}")

# ---------------- Staff access: invitations, second factor, password reset ----------------
import base64, hashlib, hmac, struct


def totp_now(secret, at=None):
    """The 6-digit RFC 6238 code an authenticator app would show."""
    key = base64.b32decode(secret + "=" * (-len(secret) % 8), casefold=True)
    counter = int((at or time.time()) // 30)
    digest = hmac.new(key, struct.pack(">Q", counter), hashlib.sha1).digest()
    offset = digest[-1] & 0x0F
    return str((struct.unpack(">I", digest[offset:offset + 4])[0] & 0x7FFFFFFF) % 1000000).zfill(6)


def mailpit_text(to):
    """The newest e-mail the API sent to an address, from the local inbox."""
    try:
        with urllib.request.urlopen(f"{MAILPIT}/api/v1/search?query=to:{to}", timeout=10) as r:
            found = json.loads(r.read()).get("messages") or []
        if not found:
            return ""
        with urllib.request.urlopen(f"{MAILPIT}/api/v1/message/{found[0]['ID']}", timeout=10) as r:
            return json.loads(r.read()).get("Text", "")
    except Exception as e:
        return f"(inbox unreachable: {e!r})"


MAILPIT = os.environ.get("QA_MAILPIT", "http://localhost:8025")

sec, _, _ = register("buyer", "Double Facteur")
s, b = sec.call("POST", "/me/mfa/totp/setup")
secret = (b or {}).get("secret", "")
check("X01", "Account", "Start setting up an authenticator app", "200 secret, otpauth URL, QR image", s == 200 and secret and str(b.get("url", "")).startswith("otpauth://") and str(b.get("qr", "")).startswith("data:image/png;base64,"), f"{s}")
s, b = sec.call("POST", "/me/mfa/totp/confirm", {"code": "000000"})
check("X02", "Account", "Confirm with a wrong code", "401 code auth.mfa_code_invalid", s == 401 and (b or {}).get("code") == "auth.mfa_code_invalid", f"{s} {b}")
s, b = sec.call("POST", "/me/mfa/totp/confirm", {"code": totp_now(secret)})
codes = (b or {}).get("recoveryCodes") or []
check("X03", "Account", "Confirm the authenticator app", "200, 10 recovery codes, a new access token", s == 200 and len(codes) == 10 and b.get("accessToken") and b["status"]["totp"] is True, f"{s} codes={len(codes)}")
sec.token = (b or {}).get("accessToken") or sec.token
s, b = sec.call("GET", "/me/mfa")
check("X04", "Account", "The session that set the factor up counts as verified", "enabled, sessionVerified", s == 200 and b.get("enabled") is True and b.get("sessionVerified") is True, f"{s} {b}")
second = Client()
s, b = second.call("POST", "/auth/login", {"email": sec.email, "password": sec.pw, "remember": True}, token=False)
mfa_token = (b or {}).get("mfaToken")
check("X05", "Account", "A right password no longer opens a session by itself", "200 mfaRequired, no access token", s == 200 and (b or {}).get("mfaRequired") is True and mfa_token and "accessToken" not in b and "totp" in b.get("methods", []), f"{s} {str(b)[:120]}")
s, b = second.call("POST", "/auth/mfa/verify", {"mfaToken": mfa_token, "code": "000000"}, token=False)
check("X06", "Account", "Second step with a wrong code", "401", s == 401, f"{s} {b}")
s, b = second.call("POST", "/auth/mfa/verify", {"mfaToken": mfa_token, "code": totp_now(secret)}, token=False)
check("X07", "Account", "Second step with the app's code", "200 access token", s == 200 and (b or {}).get("accessToken"), f"{s} {str(b)[:100]}")
s, b = second.call("POST", "/auth/mfa/verify", {"mfaToken": mfa_token, "code": totp_now(secret)}, token=False)
check("X08", "Account", "A finished second step cannot be replayed", "401", s == 401, f"{s}")
for attempt, want in ((1, 200), (2, 401)):
    s, b = second.call("POST", "/auth/login", {"email": sec.email, "password": sec.pw}, token=False)
    s, b = second.call("POST", "/auth/mfa/verify", {"mfaToken": (b or {}).get("mfaToken"), "recoveryCode": codes[0].lower()}, token=False)
    check(f"X09.{attempt}", "Account", f"Recovery code, use {attempt}", str(want), s == want, f"{s}")
s, b = sec.call("POST", "/me/mfa/totp/disable", {"password": "not-the-password"})
check("X10", "Account", "Remove the factor with a wrong password", "401", s == 401, f"{s}")
s, b = sec.call("POST", "/me/mfa/totp/disable", {"password": sec.pw})
check("X11", "Account", "Remove the authenticator app", "200 disabled, no code left", s == 200 and b.get("enabled") is False and b.get("recoveryCodes") == 0, f"{s} {b}")
s, b = sec.call("POST", "/me/mfa/passkeys/options")
check("X12", "Account", "Start registering a passkey", "200 options with a challenge and a flow token", s == 200 and (b or {}).get("flowToken") and b["options"]["publicKey"].get("challenge"), f"{s}")

invited = f"qa-invited-{uuid.uuid4().hex[:8]}@betailo.test"
s, b = sec.call("POST", "/admin/staff/invitations", {"email": invited, "role": "support"})
check("X20", "Account", "An ordinary account inviting staff", "403", s == 403, f"{s}")
s, b = adm.call("POST", "/admin/staff/invitations", {"email": invited, "role": "support"})
link = (b or {}).get("link", "")
inv_id = (b or {}).get("id")
check("X21", "Staff", "Invite a member by e-mail", "201 with a one-time link, e-mailed", s == 201 and "/invitation/" in link and b.get("mailed") is True, f"{s} {str(b)[:160]}")
check("X22", "Staff", "The invitation e-mail carries the link", "the same link in the inbox", link != "" and link in mailpit_text(invited), mailpit_text(invited)[:120])
s, b = adm.call("GET", "/admin/staff")
check("X23", "Staff", "Pending invitations are listed with the team, without their link", "the address, no link", s == 200 and any(i.get("email") == invited and not i.get("link") for i in b.get("invitations", [])), f"{s}")
s, b = adm.call("POST", f"/admin/staff/invitations/{inv_id}/resend")
old_link, link = link, (b or {}).get("link", "")
check("X24", "Staff", "Resend an invitation", "200 with a new link", s == 200 and link and link != old_link, f"{s}")
s, b = g.call("GET", "/auth/invitations/" + old_link.rsplit("/", 1)[-1], token=False)
check("X25", "Guest", "The replaced link is dead", "404 code auth.invitation_invalid", s == 404 and (b or {}).get("code") == "auth.invitation_invalid", f"{s} {b}")
inv_token = link.rsplit("/", 1)[-1]
s, b = g.call("GET", "/auth/invitations/" + inv_token, token=False)
check("X26", "Guest", "Open an invitation", "200 address, role, no account yet", s == 200 and b.get("email") == invited and b.get("role") == "support" and b.get("accountExists") is False, f"{s} {b}")
member = Client()
s, b = member.call("POST", "/auth/invitations/accept", {"token": inv_token, "displayName": "Kevin Invité", "password": "short"}, token=False)
check("X27", "Guest", "Accept with a password too short", "400", s == 400, f"{s}")
s, b = member.call("POST", "/auth/invitations/accept", {"token": inv_token, "displayName": "Kevin Invité", "password": "Qa-invited-pass-1"}, token=False)
member.token = (b or {}).get("accessToken")
check("X28", "Guest", "Accept an invitation: the account is created with its role and signed in", "200 access token, staffRole=support", s == 200 and member.token and (b.get("user") or {}).get("staffRole") == "support", f"{s} {str(b)[:140]}")
s, b = member.call("GET", "/admin/overview")
check("X29", "Staff", "The invited member reaches the back office", "200", s == 200, f"{s}")
s, b = member.call("POST", "/auth/invitations/accept", {"token": inv_token, "password": "Qa-invited-pass-1"}, token=False)
check("X30", "Guest", "An invitation works once", "404", s == 404, f"{s}")
s, b = adm.call("POST", "/admin/staff/invitations", {"email": invited, "role": "finance"})
check("X31", "Staff", "Invite someone already on the staff", "409 code admin.staff_already_member", s == 409 and (b or {}).get("code") == "admin.staff_already_member", f"{s} {b}")
other = f"qa-invited-{uuid.uuid4().hex[:8]}@betailo.test"
s, b = adm.call("POST", "/admin/staff/invitations", {"email": other, "role": "moderator"})
cancel_link, cancel_id = (b or {}).get("link", ""), (b or {}).get("id")
s, b = adm.call("DELETE", f"/admin/staff/invitations/{cancel_id}")
s2, b2 = g.call("GET", "/auth/invitations/" + cancel_link.rsplit("/", 1)[-1], token=False)
check("X32", "Staff", "Cancel an invitation", "204, then its link is dead", s == 204 and s2 == 404, f"{s} {s2}")

s, b = g.call("POST", "/auth/password/forgot", {"email": "nobody-" + uuid.uuid4().hex[:6] + "@betailo.test"}, token=False)
s2, b2 = g.call("POST", "/auth/password/forgot", {"email": sec.email}, token=False)
check("X40", "Guest", "Asking for a reset link answers the same for any address", "202 and 202", s == 202 and s2 == 202, f"{s} {s2}")
mail = mailpit_text(sec.email)
reset_token = mail.split("/reset-password/")[1].split()[0] if "/reset-password/" in mail else ""
check("X41", "Guest", "The reset link is e-mailed to the account", "a link in the inbox", bool(reset_token), mail[:120])
s, b = g.call("POST", "/auth/password/reset", {"token": reset_token, "password": "short"}, token=False)
check("X42", "Guest", "Reset with a password too short", "400", s == 400, f"{s}")
s, b = g.call("POST", "/auth/password/reset", {"token": reset_token, "password": "Qa-reset-pass-2"}, token=False)
check("X43", "Guest", "Choose a new password from the link", "204", s == 204, f"{s} {b}")
s, b = g.call("POST", "/auth/login", {"email": sec.email, "password": sec.pw}, token=False)
s2, b2 = g.call("POST", "/auth/login", {"email": sec.email, "password": "Qa-reset-pass-2"}, token=False)
check("X44", "Guest", "The old password is out, the new one works", "401 then 200", s == 401 and s2 == 200, f"{s} {s2}")
s, b = g.call("POST", "/auth/password/reset", {"token": reset_token, "password": "Qa-reset-pass-3"}, token=False)
check("X45", "Guest", "A reset link works once", "404 code auth.reset_link_invalid", s == 404 and (b or {}).get("code") == "auth.reset_link_invalid", f"{s} {b}")

# ---------- reference data (catalog) ----------
s, ref = g.call("GET", "/reference", token=False)
cats = {c["key"]: c for c in (ref or {}).get("categories", [])} if isinstance(ref, dict) else {}
check("C01", "Guest", "GET /reference: categories with their sub-categories", "200, volaille holds dindes and pondeuses",
      s == 200 and "dindes" in [x["key"] for x in cats.get("volaille", {}).get("subcategories", [])] and "pondeuses" in [x["key"] for x in cats.get("volaille", {}).get("subcategories", [])],
      f"{s} v{(ref or {}).get('version') if isinstance(ref, dict) else ref} {list(cats)}")
s, b = g.call("GET", "/listings?category=volaille&pageSize=50", token=False)
species = {it["species"] for it in (b or {}).get("items", [])} if isinstance(b, dict) else set()
check("C02", "Guest", "Filter the marketplace by a whole category", "200, only poultry", s == 200 and species <= {x["key"] for x in cats.get("volaille", {}).get("subcategories", [])}, f"{s} {sorted(species)}")
s, b = buy.call("POST", "/forum/posts", {"category": "politique", "title": "Hors sujet", "body": "Texte"})
check("C03", "Buyer", "Post under a category the catalog doesn't list", "400 forum.category_unknown", s == 400 and (b or {}).get("code") == "forum.category_unknown", f"{s} {b}")
s, b = far.call("POST", "/cycles", {"farmId": farm_id, "species": "licornes", "type": "ponte", "capacity": 10, "startsAt": "2027-01-10T00:00:00Z"})
check("C04", "Farmer", "Open a cycle of a species the catalog doesn't offer", "400 catalog.species_unknown", s == 400 and (b or {}).get("code") == "catalog.species_unknown", f"{s} {b}")

# ---------- profile photos ----------
pic, _, _ = register("avatar", "Awa Photo")
other, _, _ = register("avatar2", "Other Photo")
s, up = pic.call("POST", "/me/avatar/presign", PHOTO_SIZES)
uploaded = s == 200 and put_file(up["uploadUrl"]) == 200 and put_file(up["thumbUploadUrl"]) == 200
check("P01", "User", "Upload a profile photo and its small copy", "200 + both uploads", uploaded, f"{s} {str(up)[:80]}")
s, b = pic.call("POST", "/me/avatar/presign", {"size": 801 << 10, "thumbSize": 1000})
check("P02", "User", "Declare a profile photo over 800 KB", "400 photo.too_large", s == 400 and (b or {}).get("code") == "photo.too_large", f"{s} {b}")
s, b = pic.call("PUT", "/me/avatar", {"key": up["key"]})
check("P03", "User", "Set it as the profile photo", "200 with avatarUrl", s == 200 and bool((b or {}).get("avatarUrl")), f"{s} {str(b)[:100]}")
s, b = pic.call("GET", "/me")
check("P04", "User", "GET /me carries the photo's small copy", "avatarUrl to the .thumb.jpg", ".thumb.jpg" in (b or {}).get("avatarUrl", ""), str((b or {}).get("avatarUrl", ""))[:100])
s, b = other.call("PUT", "/me/avatar", {"key": up["key"]})
check("P05", "User", "Take someone else's uploaded photo as your own", "400", s == 400, f"{s} {b}")
s, b = pic.call("POST", "/forum/posts", {"category": "general", "title": "Photo de profil QA", "body": "Bonjour à tous, je teste la photo."})
s, b = g.call("GET", f"/forum/posts/{(b or {}).get('id')}", token=False)
check("P06", "Guest", "A forum post shows its author's public name and photo", "authorName 'Awa P.' + authorAvatarUrl",
      s == 200 and (b or {}).get("authorName") == "Awa P." and bool((b or {}).get("authorAvatarUrl")), f"{s} {(b or {}).get('authorName')}")
s, b = pic.call("DELETE", "/me/avatar")
check("P07", "User", "Remove the profile photo", "200 without avatarUrl", s == 200 and not (b or {}).get("avatarUrl"), f"{s} {str(b)[:80]}")

failed = [r for r in RESULTS if not r["ok"]]
print(f"\n{len(RESULTS) - len(failed)}/{len(RESULTS)} passed")
sys.exit(1 if failed else 0)
