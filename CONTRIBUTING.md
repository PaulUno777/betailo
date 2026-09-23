# Contribuer à Betailo

Projet personnel/portfolio — les contributions externes ne sont pas activement
recherchées, mais ce guide garde le repo cohérent au fil du développement.

## Organisation

Monorepo à deux dossiers indépendants :

- `backend/` — API Go (voir son `README` interne si présent, sinon le
  `Makefile` liste les commandes courantes : `make run`, `make test`, `make lint`).
- `frontend/` — application Angular + Tailwind (`npm start`, `npm test`, `npm run build`).

Chaque dossier a ses propres dépendances et sa propre pipeline CI (voir
`.github/workflows/ci.yml`) ; ils ne partagent pas de `node_modules`/`go.mod`.

## Avant de committer

- Backend : `make fmt && make lint && make test`
- Frontend : `npm run lint && npm test -- --watch=false && npm run build`

## Convention de commits

Commits courts, à l'impératif, préfixés par le dossier concerné quand c'est
utile : `backend: add reservation service`, `frontend: wire tailwind theme`.

## Décisions de conception

Le document de proposition (architecture, modèle de données, direction
artistique, roadmap) fait foi pour les choix déjà tranchés. Toute déviation
notable (nouvelle dépendance majeure, changement de schéma) doit y être
répercutée.
