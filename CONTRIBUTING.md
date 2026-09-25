# Contribuer à Betailo

Projet personnel/portfolio — les contributions externes ne sont pas activement
recherchées, mais ce guide garde le repo cohérent au fil du développement.

## Organisation

Monorepo à deux dossiers indépendants :

- `backend/` — API Go (voir son `README` interne si présent, sinon le
  `Makefile` liste les commandes courantes : `make run`, `make test`, `make lint`).
- `frontend/` — application Angular + Tailwind, gérée avec **pnpm**
  (`pnpm start`, `pnpm test`, `pnpm run build`).

Chaque dossier a ses propres dépendances et sa propre pipeline CI (voir
`.github/workflows/ci.yml`) ; ils ne partagent pas de `node_modules`/`go.mod`.

## Avant de committer

- Backend : `make fmt && make lint && make test`
- Frontend : `pnpm run lint && pnpm test -- --watch=false && pnpm run build`

## Convention de commits

Commits courts, à l'impératif, préfixés par le dossier concerné quand c'est
utile : `backend: add reservation service`, `frontend: wire tailwind theme`.

## Convention de branches

Une branche par item de la roadmap (voir "Betailo — Plan de développement",
section "Roadmap de développement") ou par correctif isolé, jamais de
développement direct sur `main` :

- `feat/<sujet>` — nouvelle fonctionnalité ou phase de roadmap (ex.
  `feat/auth`, `feat/farm-cycle-listing`, `feat/mongodb-migration`).
- `fix/<sujet>` — correctif qui ne rentre pas dans une phase en cours.
- `chore/<sujet>` — outillage, CI, dépendances.

Une branche = une Pull Request vers `main`, avec la CI verte avant merge
(`.github/workflows/ci.yml`). Fusionner (ou rebaser) `main` dans la branche
avant d'ouvrir la PR si `main` a avancé entre-temps.

## Langue

Le code (identifiants Go, commentaires, valeurs de chaînes y compris les
`enum` persistés) est exclusivement en anglais — voir
[`docs/adr/0005-english-only-in-code-french-in-product-docs.md`](docs/adr/0005-english-only-in-code-french-in-product-docs.md).
La documentation produit (ce fichier, le `README`, les documents dans
`docs/`) reste en français.

## Décisions de conception

Le document de proposition (architecture, modèle de données, direction
artistique, roadmap) fait foi pour les choix produit déjà tranchés. Toute
déviation notable (nouvelle dépendance majeure, changement de schéma) doit y
être répercutée.

Les décisions **techniques** ponctuelles (pourquoi telle techno plutôt
qu'une autre, à tel moment, face à telle contrainte) vont dans une ADR sous
[`docs/adr/`](docs/adr/README.md) plutôt que dans le document de
proposition — un nouveau choix qui en remplace un ancien ajoute une
nouvelle ADR qui référence celle qu'elle remplace, sans réécrire
l'historique.
