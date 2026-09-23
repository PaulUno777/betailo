-- Initial schema for Betailo — mirrors the data model in the proposal
-- document ("Modèle de données"). Kept minimal on purpose: enough to
-- start the "Socle" phase of the roadmap, extended migration by
-- migration as each module is built.
--
-- Run with golang-migrate, e.g.:
--   migrate -path migrations -database "$DATABASE_URL" up

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('eleveur', 'acheteur', 'admin')),
    is_verified   BOOLEAN NOT NULL DEFAULT FALSE,
    trust_score   NUMERIC(5, 2) NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE farms (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    location   TEXT NOT NULL,
    species    TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cycles (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    farm_id        UUID NOT NULL REFERENCES farms (id) ON DELETE CASCADE,
    species        TEXT NOT NULL,
    cycle_type     TEXT NOT NULL, -- reproduction | engraissement | ponte | ...
    started_at     DATE NOT NULL,
    expected_days  INTEGER NOT NULL,
    status         TEXT NOT NULL DEFAULT 'planifie', -- planifie | en_cours | termine
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE listings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_id        UUID NOT NULL REFERENCES cycles (id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    quantity_total  INTEGER NOT NULL CHECK (quantity_total > 0),
    quantity_left   INTEGER NOT NULL CHECK (quantity_left >= 0),
    unit_price      NUMERIC(12, 2) NOT NULL CHECK (unit_price >= 0),
    available_at    DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reservations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id  UUID NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    buyer_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    quantity    INTEGER NOT NULL CHECK (quantity > 0),
    status      TEXT NOT NULL DEFAULT 'en_attente', -- en_attente | confirmee | livree | annulee
    amount      NUMERIC(12, 2) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id   UUID NOT NULL REFERENCES reservations (id) ON DELETE CASCADE,
    amount           NUMERIC(12, 2) NOT NULL,
    status           TEXT NOT NULL DEFAULT 'retenu', -- retenu | libere | rembourse
    external_ref     TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id  UUID NOT NULL REFERENCES reservations (id) ON DELETE CASCADE,
    rating          SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_type  TEXT NOT NULL, -- listing | profile | forum_post
    target_id    UUID NOT NULL,
    reason       TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'ouvert', -- ouvert | traite | rejete
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE forum_posts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category    TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    listing_id  UUID REFERENCES listings (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE forum_replies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID NOT NULL REFERENCES forum_posts (id) ON DELETE CASCADE,
    author_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
