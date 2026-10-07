"""End-to-end API smoke test over every profile (guest, buyer, farmer, admin).

Walks the real API through each profile's flows — browsing and search,
signup and becoming a seller, the sell wizard with a photo upload,
Mobile Money checkout, the waiting list, delivery and escrow, reviews,
disputes, moderation and suspension — including the refusals (IDOR
attempts, invalid input, forbidden roles) and the error codes clients
translate. Each check prints PASS/FAIL; the exit code is the number of
failures (capped at 1), so CI fails on any.

Creates fresh accounts on every run, so it's repeatable against any
database. Needs an admin account (the API never lets one self-register):

    QA_API=http://localhost:8080          (default)
    QA_ADMIN_EMAIL=... QA_ADMIN_PASSWORD=...
    QA_RESULTS=path/to/results.json       (optional)

Run it with scripts/qa/run_smoke.sh, which starts the stack and seeds
the admin.
"""
import json, os, sys, time, uuid, urllib.request, urllib.error, http.cookiejar, datetime

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


def register(role, name=None):
    time.sleep(REGISTER_PAUSE)
    c = Client()
    email = f"qa-{role}-{uuid.uuid4().hex[:8]}@betailo.test"
    pw = "Qa-" + uuid.uuid4().hex[:12]
    s, b = c.call("POST", "/auth/register", {"email": email, "password": pw, "role": role, "displayName": name or ""}, token=False)
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
s, b = g.call("POST", "/auth/register", {"email": "not-an-email", "password": "short", "role": "buyer"}, token=False)
check("G07", "Guest", "Register with invalid email/short password", "400 with message", s == 400, f"{s} {b}")
s, b = g.call("POST", "/auth/register", {"email": f"qa-esc-{uuid.uuid4().hex[:6]}@betailo.test", "password": "Qa-longpassword1", "role": "admin"}, token=False)
check("G08", "Guest", "Self-register as admin (privilege escalation)", "refused (4xx)", s >= 400, f"{s} {b}")
s, b = g.call("POST", "/auth/login", {"email": "nobody-" + uuid.uuid4().hex[:6] + "@betailo.test", "password": "wrong-password"}, token=False)
check("G09", "Guest", "Login with wrong password", "401, generic message", s == 401, f"{s} {b}")

# ---------------- Farmer: onboarding + listing ----------------
far, s, b = register("buyer", "Awa Fermière")
check("F01", "Farmer", "Register as buyer then become seller", "201 register", s == 201, s)
s, b = far.call("POST", "/me/become-seller", {"farm": {"name": "Ferme QA", "location": "Bafoussam", "species": ["jeunes-poulets"], "contactPhone": "677001122"}})
check("F02", "Farmer", "POST /me/become-seller", "201 farm", s in (200, 201) and isinstance(b, dict) and b.get("id"), f"{s} {str(b)[:120]}")
farm_id = (b or {}).get("id") if isinstance(b, dict) else None
s, b = far.call("POST", "/auth/refresh", token=False)
far.token = (b or {}).get("accessToken", far.token)
s, me = far.call("GET", "/me")
check("F03", "Farmer", "Role is farmer after session refresh", "role=farmer", isinstance(me, dict) and me.get("role") == "farmer", f"{s} role={me.get('role') if isinstance(me, dict) else me}")
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
s, b = far.call("POST", f"/listings/{qlisting}/photos/presign")
check("F08", "Farmer", "Presign a listing photo upload", "200 key + uploadUrl", s == 200 and isinstance(b, dict) and b.get("uploadUrl"), f"{s} {str(b)[:100]}")
key = (b or {}).get("key") if isinstance(b, dict) else None
upload = (b or {}).get("uploadUrl") if isinstance(b, dict) else None
if upload:
    # 1x1 JPEG
    jpeg = bytes.fromhex("ffd8ffe000104a46494600010100000100010000ffdb004300080606070605080707070909080a0c140d0c0b0b0c1912130f141d1a1f1e1d1a1c1c20242e2720222c231c1c2837292c30313434341f27393d38323c2e333432ffc0000b080001000101011100ffc4001f0000010501010101010100000000000000000102030405060708090a0bffc400b5100002010303020403050504040000017d01020300041105122131410613516107227114328191a1082342b1c11552d1f02433627282090a161718191a25262728292a3435363738393a434445464748494a535455565758595a636465666768696a737475767778797a838485868788898a92939495969798999aa2a3a4a5a6a7a8a9aab2b3b4b5b6b7b8b9bac2c3c4c5c6c7c8c9cad2d3d4d5d6d7d8d9dae1e2e3e4e5e6e7e8e9eaf1f2f3f4f5f6f7f8f9faffda0008010100003f00fbfcffd9")
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
    s, b = far.call("PUT", f"/listings/{qlisting}/photos", {"keys": [key]})
    check("F10", "Farmer", "Attach photo keys to listing", "200", s == 200, f"{s} {str(b)[:100]}")
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
check("A01", "Admin", "Admin login", "200 role=admin", s == 200 and adm.user.get("role") == "admin", f"{s} role={adm.user.get('role')}")
for path in ["/admin/overview", "/admin/users", "/admin/listings", "/admin/escrow", "/admin/reports", "/admin/cycles"]:
    s, b = adm.call("GET", path)
    check("A02" + path, "Admin", f"GET {path}", "200", s == 200, s)
if report_id:
    s, b = adm.call("GET", f"/admin/reports/{report_id}")
    check("A03", "Admin", "Report detail with risk signals", "200 riskScore", s == 200 and "riskScore" in json.dumps(b), f"{s}")
    s, b = adm.call("POST", f"/admin/reports/{report_id}/resolve", {"action": "dismiss"})
    check("A04", "Admin", "Resolve report (dismiss)", "200", s == 200, f"{s} {str(b)[:100]}")
s, b = adm.call("POST", f"/admin/users/{far.user.get('id')}/verify")
check("A05", "Admin", "Verify a farmer", "200", s == 200, f"{s} {str(b)[:100]}")
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
wl_farmer, _, _ = register("buyer", "Waitlist Farmer")
s, wf = wl_farmer.call("POST", "/me/become-seller", {"farm": {"name": "Ferme WL", "location": "Dschang", "region": "Ouest"}})
wl_farmer.token = (wl_farmer.call("POST", "/auth/refresh", token=False)[1] or {}).get("accessToken", wl_farmer.token)
s, ql = wl_farmer.call("POST", f"/farms/{wf['id']}/quick-listings", {"species": "poussins", "quantity": 10, "unitPrice": 700, "title": "Lot liste d'attente", "deliveryAvailable": False})
wl_listing, wl_cycle = ql["listing"]["id"], ql["cycle"]["id"]
s, up = wl_farmer.call("POST", f"/listings/{wl_listing}/photos/presign")
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
failed = [r for r in RESULTS if not r["ok"]]
print(f"\n{len(RESULTS) - len(failed)}/{len(RESULTS)} passed")
sys.exit(1 if failed else 0)
