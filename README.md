# Betailo

> Plateforme communautaire et marketplace pour éleveurs — réservation par
> cycle d'élevage, avis, forum, et outils anti-arnaque, sur un socle
> **Go** (backend) + **Angular / Tailwind** (frontend).

[![CI](https://github.com/tchiopaulinjunior/betailo/actions/workflows/ci.yml/badge.svg)](https://github.com/tchiopaulinjunior/betailo/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![Angular](https://img.shields.io/badge/Angular-21-DD0031?logo=angular&logoColor=white)
![pnpm](https://img.shields.io/badge/pnpm-12-F69220?logo=pnpm&logoColor=white)
![License: MIT](https://img.shields.io/badge/license-MIT-informational)

> Projet portfolio : ce dépôt est l'amorçage technique (monorepo, CI,
> squelettes backend/frontend) du projet détaillé dans le document de
> proposition — architecture complète, modèle de données, direction
> artistique et roadmap. Ce README documente **ce qui existe déjà dans le
> code** ; le document de proposition reste la référence produit.

## Pourquoi ce projet

La plupart des plateformes d'élevage se contentent de copier un
e-commerce classique. Betailo part d'un constat différent : un éleveur ne
vend pas un "produit" statique, il gère des **cycles** (reproduction,
engraissement, ponte...) qui ont un début, une durée et une fin — et
c'est ce cycle, pas un simple stock, que l'acheteur réserve. Ce dépôt est
un projet portfolio destiné à démontrer, de façon concrète et finie,
la conception d'une API Go en couches, une modélisation de domaine non
triviale, et une interface pensée pour un public peu digitalisé.

## Sommaire

- [Structure du monorepo](#structure-du-monorepo)
- [Démarrage rapide](#démarrage-rapide)
- [Stack technique](#stack-technique)
- [Commandes utiles](#commandes-utiles)
- [Intégration continue](#intégration-continue)
- [État du projet](#état-du-projet--roadmap)
- [Licence](#licence)

## Structure du monorepo

```
betailo/
├── backend/     # API Go — voir backend/ pour le détail des modules
├── frontend/    # Application Angular + Tailwind
├── docker-compose.yml   # Stack complète (Postgres + backend + frontend)
└── .github/workflows/ci.yml
```

Deux projets indépendants dans un seul dépôt : pas de tooling partagé
(pas de Nx/Turborepo) volontairement, pour rester simple à ce stade —
chacun a son propre gestionnaire de dépendances et sa propre CI.

Détail du backend (le cœur du projet) :

```
backend/
├── cmd/api/                # point d'entrée du binaire
├── internal/
│   ├── auth/ user/ farm/ cycle/ listing/ reservation/
│   ├── payment/ review/ forum/ trust/ notification/
│   └── platform/{db, httpserver}/
├── migrations/              # schéma SQL (golang-migrate)
└── Dockerfile
```

Chaque dossier sous `internal/` correspond à un module métier décrit dans
le document de proposition ; ils ne contiennent pour l'instant qu'un
type placeholder et un commentaire de responsabilité, à charge de la
roadmap de les remplir phase par phase.

## Démarrage rapide

Prérequis : Go 1.24+, Node 22.12+ (Node ≥22.22.3, ≥24.15 ou ≥26 pour passer
à Angular 22 dès sa sortie stable — voir note plus bas), [pnpm](https://pnpm.io/)
via `corepack enable` (inclus avec Node), Docker (optionnel mais recommandé
pour Postgres).

### Avec Docker (le plus simple)

```bash
docker compose up --build
```

- Backend : http://localhost:8080/health
- Frontend : http://localhost:4200
- Postgres : localhost:5432 (user/mdp/db : `betailo`)

### En local, sans Docker

```bash
# Backend
cd backend
cp .env.example .env   # ajuster DATABASE_URL si besoin
go run ./cmd/api        # démarre sur :8080

# Frontend (dans un autre terminal)
cd frontend
corepack enable          # active pnpm (une seule fois par machine)
pnpm install
pnpm start                # démarre sur :4200
```

## Stack technique

| Couche | Choix | Détail |
| --- | --- | --- |
| Backend | Go 1.24, `net/http` natif, architecture en couches (handler → service → repository) | voir `backend/internal/platform/httpserver` |
| Base de données | PostgreSQL, migrations SQL versionnées | voir `backend/migrations/` |
| Frontend | Angular 21 (standalone components) + Tailwind CSS v4, géré avec pnpm | palette de marque déclarée dans `frontend/src/styles.css` (`@theme`) |
| CI | GitHub Actions — jobs indépendants backend/frontend | voir `.github/workflows/ci.yml` |
| Conteneurisation | Dockerfiles multi-stage (backend et frontend) + `docker-compose.yml` | |

Le raisonnement détaillé derrière chaque choix (pourquoi pas d'ORM, pourquoi
un monolithe plutôt que des microservices, etc.) est dans le document de
proposition, section "Architecture technique et stack Go".

## Commandes utiles

| Commande | Effet |
| --- | --- |
| `cd backend && make run` | Lance l'API en local |
| `cd backend && make test` | Tests Go avec race detector |
| `cd backend && make lint` | golangci-lint |
| `cd frontend && pnpm test` | Tests unitaires Angular (Karma + Chrome) |
| `cd frontend && pnpm run lint` | ESLint |
| `cd frontend && pnpm run build` | Build de production |
| `docker compose up --build` | Stack complète (Postgres + backend + frontend) |

## Intégration continue

Deux jobs indépendants tournent sur chaque push/PR vers `main` (voir
`.github/workflows/ci.yml`) :

- **Backend** : format (`gofmt`), `go vet`, tests avec race detector,
  build, et `golangci-lint`.
- **Frontend** : `pnpm install --frozen-lockfile`, ESLint, tests unitaires
  en Chrome headless, build de production.

Un job ne bloque pas l'autre : une régression frontend n'empêche pas de
valider une pull request backend, et inversement.

> **Note sur la version d'Angular.** La dernière version stable d'Angular
> au moment de l'écriture est la v22, mais elle exige Node ≥22.22.3 (ou
> ≥24.15 / ≥26). Le projet est donc sur **Angular 21** (la dernière version
> pleinement testée avec un Node 22.22.x classique) ; passer à la v22 une
> fois sur un Node suffisamment récent se fait avec
> `pnpm dlx @angular/cli@22 update @angular/core@22 @angular/cli@22`.
> Le paquet `angular-eslint` est lui déjà en v22 (pas de version stable
> intermédiaire publiée) — un avertissement cosmétique de désaccord de
> version peut apparaître au lint, sans impact réel.

## État du projet / roadmap

Ce dépôt correspond à la **phase 1 (Socle)** de la roadmap du document de
proposition : structure du monorepo, CI de base, squelette backend
compilable et testé, squelette frontend avec la charte graphique
branchée. Les phases suivantes (cœur transactionnel, confiance/forum,
asynchrone, finitions) restent à construire — voir le document de
proposition pour le détail phase par phase et ce que chacune démontre
comme compétence.

## Licence

MIT — voir [`LICENSE`](./LICENSE).
