# Betailo

> Plateforme communautaire et marketplace pour éleveurs — réservation par
> cycle d'élevage, avis, forum, et outils anti-arnaque, sur un socle
> **Go** (backend) + **Angular / Tailwind** (frontend).

![CI](https://github.com/pauluno777/betailo/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![MongoDB](https://img.shields.io/badge/MongoDB-Atlas-47A248?logo=mongodb&logoColor=white)
![Angular](https://img.shields.io/badge/Angular-21-DD0031?logo=angular&logoColor=white)
![pnpm](https://img.shields.io/badge/pnpm-12-F69220?logo=pnpm&logoColor=white)
![License: MIT](https://img.shields.io/badge/license-MIT-informational)

> Projet portfolio : ce dépôt est l'amorçage technique (monorepo, CI, squelettes backend/frontend) du projet détaillé dans le document de proposition architecture complète, modèle de données, direction artistique et roadmap. Ce README documente **ce qui existe déjà dans le code** ; le document de proposition reste la référence produit.



## Pourquoi ce projet

La plupart des plateformes d'élevage se contentent de copier un
e-commerce classique. Betailo part d'un constat différent : un éleveur ne
vend pas un "produit" statique, il gère des **cycles** (reproduction, engraissement, ponte...) qui ont un début, une durée et une fin et c'est ce cycle, pas un simple stock, que l'acheteur réserve. Ce dépôt est un projet portfolio destiné à démontrer, de façon concrète et finie, la conception d'une API Go en couches, une modélisation de domaine non triviale, et une interface pensée pour un public peu digitalisé.

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
├── docker-compose.yml   # Stack complète (MongoDB + SeaweedFS + backend + frontend)
└── .github/workflows/ci.yml
```

Deux projets indépendants dans un seul dépôt : pas de tooling partagé
(pas de Nx/Turborepo) volontairement, pour rester simple à ce stade —
chacun a son propre gestionnaire de dépendances et sa propre CI.

Détail du backend (le cœur du projet) :

```
backend/
├── cmd/api/                    # point d'entrée du binaire (juste : Initialize -> Bootstrap -> Run)
├── domain/                     # entités pures — zéro tag bson/json, zéro import d'infra
├── application/
│   ├── ports/                  # interfaces (repositories, TxManager, ObjectStorage)
│   └── usecase/reservation/    # le seul usecase avec une vraie logique métier pour l'instant
├── infrastructure/
│   ├── mongodb/                 # implémente application/ports contre MongoDB (mongo-driver/v2)
│   └── objectstorage/           # implémente application/ports.ObjectStorage (S3-compatible)
├── interfaces/httpserver/       # couche de livraison : routeur, middlewares, JSON
├── internal/app/                # composition root : connecte tout ce qui précède
├── schemas/                     # un validateur $jsonSchema MongoDB par collection
└── Dockerfile
```

Architecture en couches à sens unique — `interfaces` → `application` →
`domain`, et `infrastructure` implémente `application/ports` sans que
`domain`/`application` ne la connaissent jamais dans l'autre sens
(Dependency Inversion) :

- **`domain/`** ne contient que des structs Go simples (types
  d'identifiants MongoDB `bson.ObjectID` mis à part — imposé par le
  modèle de données du document "Concepts & Architecture" — aucune
  dépendance à `mongo-driver`, `net/http` ou quoi que ce soit d'autre).
- **`application/ports/`** déclare les interfaces dont les usecases ont
  besoin (`ListingRepository`, `CycleRepository`, `ReservationRepository`,
  `TxManager`, `ObjectStorage`), chacune réduite aux méthodes réellement
  utilisées (Interface Segregation) — les ajouter ici, pas dans
  `infrastructure/`, c'est ce qui permet à `application/usecase` de ne
  jamais importer `mongo-driver`.
- **`application/usecase/reservation`** implémente le flux "1.
  Réservation d'un cycle, sans survente" : `ReserveCycle.Execute` ne
  dépend que des interfaces ci-dessus, jamais de `*mongo.Client` — il se
  teste avec des doublons (voir `reserve_cycle_test.go`), sans base de
  données.
- **`infrastructure/mongodb`** est le seul package qui importe
  `mongo-driver` ; chaque repository y implémente une interface de
  `application/ports` et convertit entre les entités `domain` (sans
  tags) et un document BSON privé au fichier (`reservationDoc`, etc.).
- **`internal/app`** est le composition root : le seul endroit qui
  connaît à la fois les usecases et MongoDB/le stockage objet, et qui
  les assemble (`Initialize`), avant que `cmd/api/main.go` n'appelle
  `Bootstrap` puis `Run`.

Les modules qui n'ont pas encore de logique métier (user, farm, review,
forum, trust/report) restent volontairement de simples entités dans
`domain/` sans port ni repository — leur en ajouter un sans méthode
réelle serait de l'abstraction prématurée ; ils suivront ce même patron
dès qu'une vraie règle métier apparaît.

## Démarrage rapide

Prérequis : Go 1.25+, Node 22.12+ (Node ≥22.22.3, ≥24.15 ou ≥26 pour passer à Angular 22 dès sa sortie stable voir note plus bas), [pnpm](https://pnpm.io/)
via `corepack enable` (inclus avec Node), Docker (optionnel mais recommandé
pour MongoDB et le stockage objet).

### Avec Docker (le plus simple)

```bash
docker compose up --build
```

- Backend : [http://localhost:8080/health](http://localhost:8080/health)
- Frontend : [http://localhost:4200](http://localhost:4200)
- MongoDB : localhost:27017 (base `betailo`). Pour Compass, mongosh ou tout
  outil lancé **depuis la machine hôte** (pas depuis un conteneur), utilisez
  `mongodb://localhost:27017/?directConnection=true` : `mongo` tourne en
  replica set à un seul nœud (nécessaire pour les transactions) et
  s'annonce en interne comme `mongo:27017`, un nom qui ne se résout que
  depuis le réseau docker-compose — sans `directConnection=true`, la
  connexion initiale réussit puis échoue en essayant de résoudre ce nom.
- SeaweedFS (stockage objet S3-compatible, dev uniquement) : API S3 sur localhost:9000 — remplace MinIO ici car ses images Docker Hub (`minio/minio`, `minio/mc`) exigent désormais une connexion Docker Hub ; même API S3, même code `ports.ObjectStorage`



### En local, sans Docker

```bash
# Backend
cd backend
cp .env.example .env   # ajuster MONGODB_URI/STORAGE_* si besoin
go run ./cmd/api        # démarre sur :8080

# Frontend (dans un autre terminal)
cd frontend
corepack enable          # active pnpm (une seule fois par machine)
pnpm install
pnpm start                # démarre sur :4200
```

Sans Docker, le backend a quand même besoin d'un MongoDB et d'un
stockage S3-compatible atteignables (il se connecte et fait tourner le
bootstrap au démarrage) : `docker compose up mongo seaweedfs`
lance juste ces deux-là si vous préférez faire tourner `go run` en
dehors de Docker.



## Stack technique


| Couche           | Choix                                                                                                    | Détail                                                                |
| ---------------- | --------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| Backend          | Go 1.25, `net/http` natif, architecture en couches (`domain` → `application` → `infrastructure`/`interfaces`), ports + inversion de dépendance (voir `application/usecase/reservation`) | voir `backend/interfaces/httpserver` et `backend/application/usecase/reservation` |
| Base de données  | MongoDB (Atlas en prod, conteneur `mongo` en dev), driver officiel `mongo-driver/v2`, aucun ODM, validateurs `$jsonSchema` appliqués au démarrage | voir `backend/infrastructure/mongodb` et `backend/schemas/`             |
| Stockage fichiers | Interface `ports.ObjectStorage` (URLs signées), implémentée via `aws-sdk-go-v2` — Cloudflare R2 en prod, SeaweedFS en dev, même code | voir `backend/infrastructure/objectstorage`                               |
| Frontend         | Angular 21 (standalone components) + Tailwind CSS v4, géré avec pnpm                                       | palette de marque déclarée dans `frontend/src/styles.css` (`@theme`) |
| CI               | GitHub Actions — jobs indépendants backend/frontend                                                        | voir `.github/workflows/ci.yml`                                      |
| Conteneurisation | Dockerfiles multi-stage (backend et frontend) + `docker-compose.yml`                                       |                                                                        |


Le raisonnement détaillé derrière chaque choix (pourquoi pas d'ORM, pourquoi
un monolithe plutôt que des microservices, pourquoi MongoDB plutôt que
relationnel, etc.) est dans le document "Concepts & Architecture".

## Commandes utiles


| Commande                        | Effet                                          |
| ------------------------------- | ---------------------------------------------- |
| `cd backend && make run`        | Lance l'API en local                           |
| `cd backend && make test`       | Tests Go avec race detector                    |
| `cd backend && make lint`       | golangci-lint                                  |
| `cd frontend && pnpm test`      | Tests unitaires Angular (Karma + Chrome)       |
| `cd frontend && pnpm run lint`  | ESLint                                         |
| `cd frontend && pnpm run build` | Build de production                            |
| `docker compose up --build`     | Stack complète (MongoDB + SeaweedFS + backend + frontend) |




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

MIT — voir `[LICENSE](./LICENSE)`.