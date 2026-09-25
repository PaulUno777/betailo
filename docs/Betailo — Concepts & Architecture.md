# Betailo — Concepts & Architecture

Sep 24, 2026 · @propose me a goland complete application to develop a an interme

## Pourquoi ce document

Le [document de proposition](https://claude.ai/artifact/1Rn28fFUoeUzRyWQT8zsin) decrit **quoi** construire et **pourquoi** (perimetre, fonctionnalites, stack, roadmap). Le [plan de developpement](https://claude.ai/artifact/F6hqrdgwqCHMHoVsNve5QF) decrit **quand** et **comment le demontrer** (phases, competences, tests, entretien).

Ce document repond a une troisieme question : **comment ca marche, concretement ?** Il explique les concepts du domaine (ce qu'est un "Cycle", pourquoi Betailo n'est pas un e-commerce classique) et montre, schemas a l'appui, comment les composants techniques s'articulent, des couches de l'API jusqu'aux conteneurs Docker. C'est le document a lire avant de plonger dans le code, ou a montrer en entretien pour expliquer une decision d'architecture en 30 secondes plutot qu'en 10 minutes.

## Vue d'ensemble de l'architecture

Betailo suit une architecture **monolithique modulaire** : un seul binaire Go, organise en couches et en modules metier, plutot qu'une multiplication de microservices que le perimetre ne justifie pas. C'est un choix assume et explique dans le document de proposition ; ce schema montre comment les blocs s'assemblent concretement.

```mermaid
flowchart TB
    subgraph Client["Client"]
        A["Angular SPA (standalone components)\nTailwind CSS"]
    end

    subgraph API["API Go (net/http natif)"]
        direction TB
        H["Handlers HTTP\n(routing, validation, JSON)"]
        S["Services\n(règles métier : Cycle, Reservation, Trust...)"]
        R["Repositories\n(accès données via mongo-driver)"]
        H --> S --> R
        MW["Middlewares\n(logging slog, auth JWT, recover)"]
        MW -.-> H
    end

    subgraph Async["Traitement asynchrone"]
        W["Worker (goroutine)\nnotifications, transitions de statut"]
    end

    DB[("MongoDB Atlas\ncollections + validateurs $jsonSchema")]
    ST[("Stockage objet\nCloudflare R2 / MinIO\nURLs signées")]

    A -- "HTTPS / JSON\nBearer JWT" --> H
    R --> DB
    S -. "enqueue" .-> W
    W --> DB
    S -. "génère une URL signée" .-> ST
    A -. "upload/téléchargement direct" .-> ST

    style Client fill:#eef6ff,stroke:#3b82f6
    style API fill:#f0fdf4,stroke:#16a34a
    style Async fill:#fff7ed,stroke:#ea580c
    style DB fill:#fdf4ff,stroke:#a855f7
    style ST fill:#fefce8,stroke:#ca8a04
```

Ce qu'il faut retenir de ce schema :

- **Le sens des dependances est a sens unique** : handler depend de service, service depend de repository — jamais l'inverse. Un repository ne connait rien du HTTP ; un handler ne touche jamais SQL directement. C'est ce qui rend chaque couche testable isolement (voir le plan de tests dans le document "Plan de developpement").
- **Les services ne se parlent pas directement entre modules.** Par exemple, le service Reservation ne modifie pas l'etat d'un Cycle en appelant le code du module Cycle : il passe par l'interface publique de ce module. Ca garde chaque module remplacable et compris independamment — utile en entretien pour expliquer une architecture "modulith".
- **Le worker asynchrone tourne dans le meme processus** (une ou plusieurs goroutines qui consomment une file en memoire ou une table "outbox" en base), pas dans un service separe : simple a deployer, tout en demontrant qu'on sait separer le travail synchrone (repondre vite a l'utilisateur) de l'asynchrone. Il a deux responsabilites concretes : envoyer les notifications (email, alerte) et faire progresser automatiquement le statut d'un Cycle dans sa machine a etats (`planifie -> actif -> termine`, ou `annule`), par exemple demarrer un cycle a sa date de debut ou expirer une reservation non payee apres un delai. C'est un exemple d'asynchrone bien plus parlant qu'un simple envoi d'email : il touche directement a la coherence des donnees metier.
- **Aucun ODM.** Le driver officiel `mongo-driver` et des requêtes explicites (`Find`, `UpdateOne`, `Aggregate`) dans les repositories, sans couche d'abstraction supplémentaire type `mgm` : la même philosophie que pour du SQL explicite, transposée au document — la requête reste lisible et son coût visible.
- **Le stockage de fichiers est isolé derrière une interface** (`storage.Provider`, deux méthodes : `PresignUpload` / `PresignDownload`). Le fournisseur concret (Cloudflare R2 en production, MinIO en local) est un détail d'implémentation injecté au démarrage — un exemple concret d'inversion de dépendance plutôt qu'un principe abstrait.

Correspondance avec le code : `cmd/api` (point d'entrée) -> `internal/platform/httpserver` (handlers + middlewares) -> `internal/<module>` (services + logique métier) -> `internal/platform/db` (connexion MongoDB, bootstrap des collections) et `internal/platform/storage` (URLs signées pour les fichiers).

## Les concepts du domaine, expliques simplement

Un e-commerce classique vend des **produits** : un stock fixe, un prix, on achete, le stock diminue. Betailo ne vend pas des produits, il donne acces a des **cycles d'elevage** : un evenement qui a un debut, une duree, une fin, et qui peut echouer ou reussir en cours de route. Comprendre cette difference explique presque toutes les decisions de modelisation ci-dessous.

### Cycle — le concept pivot

Un **Cycle** represente une periode d'elevage precise : une couvee de poussins, un lot de porcelets en engraissement, une periode de ponte, une gestation. Il a un statut qui evolue dans le temps (`planifie -> en cours -> termine` ou `annule`), une capacite (nombre de places / d'unites disponibles) et des dates cles. C'est l'entite **pivot** : tout le reste s'accroche a un Cycle plutot qu'a un "produit" statique.

Pourquoi ce choix plutot qu'un modele e-commerce standard : un acheteur qui reserve un cycle de couvee reserve en realite un **engagement futur et incertain** (le taux d'eclosion n'est jamais garanti a 100%), pas un objet deja en stock. Modeliser ça comme un "produit" cacherait cette incertitude et empecherait de representer proprement des notions comme la survente (voir plus bas) ou le suivi sanitaire en cours de cycle.

### Listing — l'offre visible

Un **Listing** est la fiche publique qu'un eleveur publie pour un Cycle : titre, description, race/espece, prix, photos. Un Cycle peut avoir un Listing (ce que les acheteurs voient) ; separer les deux permet par exemple de preparer un Cycle avant de le publier, ou de changer la presentation sans toucher aux donnees zootechniques.

### Reservation — la reponse a la rareté

Une **Reservation** engage une place ou une quantite sur un Cycle donne, avant que le cycle ne se termine. Le point delicat, et une des demonstrations techniques les plus interessantes du projet : eviter la **survente** (deux acheteurs qui reservent la derniere place en meme temps). C'est traite au niveau base de donnees (transaction + verrou ou contrainte, detaille dans le diagramme de sequence plus bas), pas seulement dans le code Go — la robustesse doit survivre a une charge concurrente.

### Trust & anti-scam — la confiance comme fonctionnalite de premier ordre

Dans un marche agricole informel, l'arnaque ("brouteur") est un risque concret et cite par les utilisateurs eux-memes. Le module **Trust** n'est pas une case a cocher secondaire : c'est un ensemble de signaux combines — verification de profil, historique de transactions reussies, avis verifies (uniquement apres une reservation reelle, jamais librement postes), signalement communautaire, et regles simples de detection (compte tres recent + prix anormalement bas + demande de paiement hors plateforme = signal fort). Le principe general : rendre visible ce qui rassure (badges, anciennete, avis verifies) et rendre couteux ce qui arnaque (delai avant premier retrait, revue manuelle des signalements repetes).

### Payment / Escrow — proteger les deux parties

Le paiement passe par une logique de type **sequestre (escrow)** simplifiee : l'argent de l'acheteur est marque "retenu" des la reservation, puis libere a l'eleveur seulement apres confirmation (livraison ou fin de cycle sans litige). Cote portfolio, c'est l'occasion de montrer une machine a etats explicite plutot qu'un simple champ booleen "paye" : `en_attente -> retenu -> libere`, avec deux issues alternatives possibles a tout moment avant liberation, `echoue` (le paiement lui-meme n'aboutit pas) et `rembourse` (litige ou annulation apres retenue).

### Forum — la couche communautaire

Le forum (posts + reponses) est volontairement simple dans le MVP : il sert a batir de la confiance et du contenu (questions d'elevage, retours d'experience) autour de la marketplace, pas a etre un forum generaliste. Il partage le meme systeme de signalement que le module Trust plutot que d'avoir sa propre moderation isolee.

### Comment ces concepts s'articulent

```mermaid
flowchart LR
    Farm["Farm (eleveur)"] -->|possede| Cycle
    Cycle -->|publie via| Listing
    Listing -->|declenche| Reservation
    Reservation -->|genere| Payment["Payment (escrow)"]
    Reservation -->|apres cloture| Review["Review (avis verifie)"]
    Farm -.->|score alimente par| Trust["Trust score"]
    Review -.->|alimente| Trust
    Reservation -.->|signalement possible| Trust
    Farm -->|participe au| Forum
```

La lecture a retenir : **Cycle est au centre**, tout le reste (Listing, Reservation, Payment, Review) en decoule ou y fait reference ; Trust est transversal, alimente par plusieurs sources plutot que par un seul module.

## Modele de donnees

Structure des collections MongoDB, avec les relations par référence (`ObjectId`) et les sous-documents embarqués :

```mermaid
erDiagram
    USERS ||--o{ FARMS : possede
    USERS ||--o{ RESERVATIONS : passe
    USERS ||--o{ REVIEWS : redige
    USERS ||--o{ REPORTS : signale
    USERS ||--o{ FORUM_POSTS : publie

    FARMS ||--o{ CYCLES : gere

    CYCLES ||--o| LISTINGS : publie_via

    LISTINGS ||--o{ RESERVATIONS : declenche

    RESERVATIONS ||--o| REVIEWS : cloture_par

    USERS {
        objectId id PK
        string email UK
        string passwordHash
        string role
        date createdAt
    }
    FARMS {
        objectId id PK
        objectId ownerId FK
        string name
        string location
        date createdAt
    }
    CYCLES {
        objectId id PK
        objectId farmId FK
        string species
        string type
        string status
        int capacity
        int availableQuantity "calcule, non stocke"
        date startsAt
        date endsAt
        array healthLog "embarque : type, description, date"
    }
    LISTINGS {
        objectId id PK
        objectId cycleId FK
        string title
        string description
        numeric unitPrice
        string status
        array photos "cles d'objets, URLs signees a la demande"
    }
    RESERVATIONS {
        objectId id PK
        objectId listingId FK
        objectId buyerId FK
        int quantity
        numeric unitPrice
        numeric totalAmount
        string status
        date createdAt
        object payment "embarque : status, amount, heldAt, releasedAt"
    }
    REVIEWS {
        objectId id PK
        objectId reservationId FK
        objectId authorId FK
        int rating
        string comment
    }
    REPORTS {
        objectId id PK
        objectId reporterId FK
        objectId targetId
        string targetType
        string reason
        string status
    }
    FORUM_POSTS {
        objectId id PK
        objectId authorId FK
        string title
        string body
        date createdAt
        array replies "embarque : authorId, body, createdAt"
    }
```

Quatre choix de modélisation spécifiques au document à remarquer :

- **`LISTINGS.cycleId` est une référence optionnelle** vers `CYCLES` : un Cycle peut exister avant d'être publié (pas encore de Listing). Contrairement à une base relationnelle, rien n'empêche techniquement de créer un Listing orphelin — cette cohérence-là est garantie par le service applicatif, pas par le schéma.
- **Trois relations bornées sont embarquées plutôt que référencées** : `healthLog` dans `CYCLES`, `payment` dans `RESERVATIONS`, `replies` dans `FORUM_POSTS`. Dans les trois cas, le sous-document est toujours lu et écrit avec son parent, et sa taille reste bornée (pas de carnet sanitaire ou de fil de discussion illimité) — exactement le cas d'usage que l'embarquement MongoDB cible. À l'inverse, `RESERVATIONS` reste une collection séparée de `LISTINGS` : c'est l'entité à plus fort volume d'écriture, et celle qui a besoin d'une mise à jour atomique indépendante (voir le diagramme de séquence anti-survente ci-dessous).
- **`RESERVATIONS.unitPrice` et `totalAmount` figent le prix au moment de la réservation**, comme une ligne de facture : si l'éleveur modifie le prix de son Listing ensuite, l'historique ne se recalcule pas rétroactivement. À l'inverse, `CYCLES.availableQuantity` n'est jamais recalculé à la volée côté application : c'est un compteur maintenu par décrémentation atomique (`findOneAndUpdate` avec `$inc`), la seule façon d'éviter la survente sans verrou explicite.
- **`REPORTS` utilise `targetId` / `targetType`** pour signaler indifféremment une annonce, un profil ou un message. En relationnel, ce choix coûtait une contrainte d'intégrité référentielle gérée côté application plutôt que par la base ; en MongoDB la question ne se pose même plus, puisqu'aucune collection n'impose de clé étrangère — un des rares cas où changer de base de données rend un compromis existant caduc plutôt que d'en créer un nouveau.

## Diagrammes de sequence des flux critiques

### 1. Reservation d'un cycle, sans survente

Le flux le plus sensible du MVP : deux acheteurs peuvent cliquer "reserver" au meme instant sur les dernieres places disponibles. La protection se joue en base de donnees, dans une transaction, pas seulement dans le code Go.

```mermaid
sequenceDiagram
    actor U as Acheteur
    participant H as Handler HTTP
    participant S as Service Reservation
    participant DB as MongoDB Atlas

    U->>H: POST /reservations {listingId, quantity}
    H->>S: CreerReservation(ctx, input)
    S->>DB: startSession + startTransaction
    S->>DB: findOneAndUpdate(cycles, {_id: cycleId, availableQuantity: {$gte: quantity}}, {$inc: {availableQuantity: -quantity}})
    alt document trouve (capacite suffisante)
        DB-->>S: cycle mis a jour
        S->>DB: insertOne(reservations, {..., unitPrice, totalAmount, status: "confirmed"})
        S->>DB: commitTransaction
        DB-->>S: ok
        S-->>H: Reservation creee
        H-->>U: 201 Created
    else aucun document (capacite insuffisante)
        DB-->>S: null
        S->>DB: abortTransaction
        S-->>H: erreur CapaciteInsuffisante
        H-->>U: 409 Conflict
    end
```

Le `findOneAndUpdate` avec la condition `availableQuantity: {$gte: quantity}` est atomique au niveau du document : si deux acheteurs réservent en même temps les dernières places, MongoDB garantit qu'une seule des deux opérations verra la condition satisfaite — pas besoin de verrou explicite comme en SQL. La transaction multi-documents (`startSession`/`commitTransaction`, supportée sur le tier gratuit Atlas car toujours un replica set) garantit que la décrémentation de capacité et la création de la réservation réussissent ou échouent ensemble.

### 2. Authentification (login + requete protegee)

```mermaid
sequenceDiagram
    actor U as Utilisateur
    participant H as Handler /auth/login
    participant S as Service Auth
    participant DB as MongoDB Atlas
    participant MW as Middleware JWT

    U->>H: POST /auth/login {email, password}
    H->>S: Authentifier(email, password)
    S->>DB: findOne(users, {email})
    DB-->>S: passwordHash
    S->>S: bcrypt.Compare(password, passwordHash)
    S-->>H: token JWT signe (exp courte + refresh)
    H-->>U: 200 {access_token, refresh_token}

    Note over U,MW: Requete protegee ulterieure
    U->>MW: GET /reservations/me (Authorization: Bearer ...)
    MW->>MW: verifier signature + expiration du JWT
    alt token valide
        MW->>H: requete + userId dans le contexte
        H-->>U: 200 donnees
    else token invalide/expire
        MW-->>U: 401 Unauthorized
    end
```

### 3. Signalement anti-scam et reaction automatique

```mermaid
sequenceDiagram
    actor U as Utilisateur
    participant H as Handler /reports
    participant S as Service Trust
    participant DB as MongoDB Atlas
    participant W as Worker async

    U->>H: POST /reports {targetId, targetType, reason}
    H->>S: Signaler(ctx, input)
    S->>DB: insertOne(reports, {...})
    S->>DB: countDocuments(reports, {targetId, createdAt: {$gte: il y a 7 jours}})
    DB-->>S: nombre de signalements
    alt seuil depasse (ex. 3 signalements en 7 jours)
        S->>DB: updateOne(listings/users, {_id: targetId}, {$set: {status: "sous_revue"}})
        S->>W: enqueue notification moderateur
        W-->>W: envoie email/notification (asynchrone)
    else sous le seuil
        S->>DB: rien de plus (signalement enregistre)
    end
    S-->>H: signalement enregistre
    H-->>U: 202 Accepted
```

Le choix de traiter la notification au moderateur **de facon asynchrone** (via le worker) plutot que dans la requete HTTP evite de ralentir la reponse a l'utilisateur qui signale, et illustre concretement la separation synchrone/asynchrone deja mentionnee dans la vue d'ensemble.

## Architecture de deploiement

Correspond directement au `docker-compose.yml` du depot : trois conteneurs, un reseau interne, un volume persistant pour les donnees.

```mermaid
flowchart TB
    subgraph Cloud["Services manages (production)"]
        ATLAS[("MongoDB Atlas\ntier gratuit M0")]
        R2[("Cloudflare R2\nstockage objet")]
    end

    subgraph Hote["Machine hote / serveur (ou docker-compose en local)"]
        subgraph Reseau["Reseau Docker interne (betailo-net)"]
            FE["conteneur frontend\nnginx:1.27-alpine\nsert le build Angular statique\nport 80"]
            BE["conteneur backend\ndistroless + binaire Go\nport 8080"]
            MONGO["conteneur mongo (dev uniquement)\nmongo:7\nport 27017"]
            MINIO["conteneur minio (dev uniquement)\nport 9000"]
        end
        VOL[("volume mongo_data\n(persistance locale)")]
    end

    Nav["Navigateur"] -->|":4200 / :80"| FE
    FE -->|"proxy /api vers"| BE
    BE -->|"MONGODB_URI (dev)"| MONGO
    BE -.->|"MONGODB_URI (prod)"| ATLAS
    BE -->|"STORAGE_ENDPOINT (dev)"| MINIO
    BE -.->|"STORAGE_ENDPOINT (prod)"| R2
    MONGO --- VOL

    style FE fill:#eef6ff,stroke:#3b82f6
    style BE fill:#f0fdf4,stroke:#16a34a
    style MONGO fill:#fdf4ff,stroke:#a855f7
    style MINIO fill:#fefce8,stroke:#ca8a04
    style Cloud fill:#f8fafc,stroke:#64748b,stroke-dasharray: 4 4
```

Points d'implementation a noter :

- **Build multi-stage** pour les deux images applicatives : le backend compile avec l'image `golang` complete puis copie uniquement le binaire dans une image `distroless` finale (surface d'attaque minimale, image legere) ; le frontend compile avec `node:22-alpine` + pnpm puis copie seulement `dist/frontend/browser` dans une image `nginx:1.27-alpine`. Aucune des deux images finales ne contient de toolchain de build.
- **Le seul état qui a besoin d'un volume local est celui du conteneur `mongo` de développement.** En production, MongoDB Atlas et Cloudflare R2 sont des services managés : aucun volume à gérer, aucune sauvegarde à scripter soi-même. Les conteneurs backend et frontend restent stateless et peuvent être recréés/redéployés sans perte de données.
- **La CI (`.github/workflows/ci.yml`) est separee du deploiement** : deux jobs independants (Go / Angular) valident chaque push, mais aucun n'implique encore de build/push d'image Docker automatise — c'est identifie comme une extension naturelle post-MVP (image registry + job de deploiement), deja notee dans la feuille de route post-MVP du document "Plan de developpement".
- **Le même code pointe vers des services différents selon l'environnement**, uniquement via les variables `MONGODB_URI` et `STORAGE_ENDPOINT` : MinIO et un Mongo local en développement, Atlas et R2 en production. Aucune branche `if environment == "prod"` dans le code applicatif — c'est la configuration qui change, pas la logique.

```mermaid
flowchart LR
    Dev["Push / Pull Request"] --> CI{"GitHub Actions"}
    CI --> JB["Job Backend\ngofmt, vet, test -race,\nbuild, golangci-lint"]
    CI --> JF["Job Frontend\npnpm install, lint,\ntests headless, build prod"]
    JB --> OK{"Les deux jobs passent ?"}
    JF --> OK
    OK -->|oui| Merge["Pret pour merge / deploiement"]
    OK -->|non| Fix["Corriger avant merge"]
```

Les deux jobs sont independants (`working-directory` distinct, pas d'etape partagee) : une regression frontend ne bloque pas la validation d'une pull request purement backend, et inversement — un choix deliberement documente dans le README du depot.
