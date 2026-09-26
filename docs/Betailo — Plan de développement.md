# Betailo — Plan de développement

Sep 24, 2026 · @propose me a goland complete application to develop a an interme

Ce document rassemble tout ce qui relève du **planning** du projet Betailo (roadmap MVP, plan d'évaluation, feuille de route au-delà du MVP) — séparé du document de proposition principal, qui reste la référence produit/architecture.

## Roadmap de développement (MVP) et compétences démontrées

| Phase | Contenu | Compétences Go mises en avant |
| --- | --- | --- |
| 1. Socle | Projet Go, `docker-compose` (MongoDB local + MinIO), bootstrap des collections/index, module `user` + `auth` (JWT, rôles), tests unitaires du service auth | Architecture en couches, sécurité de base, tests |
| 2. Cœur métier | Modules `farm`, `cycle`, `listing` : CRUD complet, règles de validation (dates, quantités) | Modélisation d'un domaine métier, validation |
| 3. Transactionnel | Module `reservation` (gestion du stock du cycle sans survente, transaction MongoDB atomique), `payment` en mode sandbox/escrow simplifié (embarqué dans la réservation) | Transactions MongoDB multi-documents, gestion de la concurrence (mise à jour atomique conditionnelle), tests d'intégration |
| 4. Confiance & communauté | `review`, `trust` (score de confiance, signalement), `forum` basique | Calculs dérivés, modération, relations parent/enfant |
| 5. Asynchrone & finitions | Worker de notifications (goroutines/queue), alertes de fin de cycle, documentation OpenAPI, durcissement (rate limiting) | Concurrency Go (goroutines, channels), observabilité |
| 6. Packaging portfolio | README détaillé avec captures/diagrammes, jeu de données de démo (seed), déploiement Docker documenté, courte vidéo de démo | Communication technique, capacité à livrer un produit fini |

Ce découpage priorise volontairement le cœur transactionnel (phases 1-3) avant les fonctionnalités périphériques (forum, avancées anti-scam) : si le temps manque, le projet reste démontrable et cohérent dès la phase 3.

**Ce que l'ensemble du projet démontre** pour un profil intermédiaire : conception d'API REST propre, modélisation de document avec règles métier non triviales, gestion de la concurrence en Go, sécurité applicative (auth, anti-fraude, rate limiting), tests à plusieurs niveaux, et capacité à livrer un produit borné et présentable — exactement ce qu'un portfolio doit prouver au-delà du "j'ai suivi un tutoriel".

## Plan d'évaluation

Le projet sert un double objectif (portfolio + montée en compétences Go) : ce plan donne des critères concrets pour vérifier, à chaque étape, que le code produit démontre vraiment un niveau intermédiaire — plutôt que de se fier à l'impression "ça a l'air fini".

### 1. Grille de compétences visées

| Compétence | Ce qui doit être visible dans le code | Où le vérifier |
| --- | --- | --- |
| Conception d'API REST | Endpoints cohérents, codes HTTP corrects, DTOs validés, erreurs structurées (pas de `panic` qui remonte au client) | `interfaces/httpserver` + tests d'intégration HTTP |
| Modélisation de domaine | Le `Cycle` pilote vraiment le modèle (rien n'est vendable hors d'un cycle), validateurs de schéma cohérents (`$jsonSchema` par collection, choix d'embarquement vs référence justifiés) | `schemas/`, revue du schéma |
| Concurrency Go | Mise à jour atomique (`findOneAndUpdate` + transaction MongoDB) pour éviter la survente sur `reservation`, worker de notifications en goroutines sans fuite ni race | `go test -race`, code de `application/usecase/reservation` |
| Sécurité applicative | JWT correctement vérifié, rate limiting actif, mots de passe hashés (jamais en clair, même en logs) | `application/usecase/auth`, `infrastructure/jwt`, `infrastructure/security`, `interfaces/httpserver` |
| Anti-scam / confiance | Score de confiance calculé à partir de données réelles (pas une constante), signalement traçable de bout en bout | `internal/trust` |
| Tests | Au moins un test d'intégration par module critique (`reservation`, `payment`, `trust`), pas seulement des tests unitaires triviaux | CI (job backend) |
| Frontend | Respect réel de la direction artistique (palette, tailles de police, zones cliquables), pas juste "ça compile" | Comparaison écran par écran avec la section "Direction artistique" |
| DevOps de base | CI verte en continu, Docker qui démarre la stack complète sans étape manuelle cachée | `.github/workflows/ci.yml`, `docker compose up` |

Un niveau "intermédiaire" convaincant, c'est cette grille majoritairement cochée sur les 3-4 premières lignes (API, modélisation, concurrency, sécurité) — le reste (forum, DevOps avancé) peut rester plus simple sans affaiblir l'ensemble.

### 2. Definition of Done par phase (roadmap)

**Phase 1 — Socle**

- [x] Monorepo, CI de base, README
- [x] Module `auth` réellement fonctionnel (pas juste un placeholder) : inscription, connexion, JWT vérifié par middleware
- [x] Au moins un test d'intégration HTTP (au-delà du `/health`)

**Phase 2 — Cœur métier**

- [x] CRUD complet `farm` / `cycle` / `listing` avec validation des règles (dates, quantités)
- [x] Un cycle sans annonce ne casse rien ; une annonce sans cycle est impossible

**Phase 3 — Transactionnel**

- [ ] Réservation qui empêche la survente sous accès concurrents (testé avec plusieurs requêtes simultanées)
- [ ] Paiement en mode séquestre simulé avec les 3 statuts (retenu/libéré/remboursé)

**Phase 4 — Confiance & communauté**

- [ ] Score de confiance recalculé à partir de vraies données (avis, cycles menés à terme)
- [ ] Signalement visible côté modération (même simple)

**Phase 5 — Asynchrone & finitions**

- [ ] Worker de notifications qui tourne sans bloquer l'API
- [ ] Rate limiting actif sur les routes sensibles (inscription, connexion)

**Phase 6 — Packaging portfolio**

- [ ] README à jour avec captures d'écran réelles (pas de texte seul)
- [ ] Démo (vidéo courte ou lien live) accessible sans setup de la part du lecteur

### 3. Plan de tests

| Niveau | Cible | Outil |
| --- | --- | --- |
| Unitaire | Logique pure (calcul du score de confiance, validation des règles de cycle) | `testing` + `testify` |
| Intégration backend | `reservation`, `payment`, `trust` contre une vraie instance MongoDB | `testcontainers-go` |
| Concurrence | Réservations simultanées sur un même listing (vérifier l'absence de survente) | `go test -race` + scénario avec plusieurs goroutines clientes |
| Frontend | Rendu des composants critiques (fiche annonce, flow de réservation) | Karma/Jasmine (déjà en place) |
| Bout en bout (optionnel) | Un parcours complet réservation → paiement simulé | à évaluer selon le temps disponible ; pas indispensable pour un portfolio |

Objectif réaliste : pas de pourcentage de couverture global à viser, mais **zéro module critique sans test d'intégration** — plus parlant en entretien qu'un chiffre de couverture global.

### 4. Préparation à la démonstration / entretien

Ordre de démo suggéré (5-7 minutes) : (1) créer un cycle côté éleveur, (2) publier une annonce liée, (3) réserver côté acheteur et montrer que le stock du cycle diminue, (4) montrer le score de confiance et un signalement, (5) montrer la CI verte sur GitHub.

Questions à anticiper (avec la réponse déjà argumentée dans le document de proposition) :

- *Pourquoi pas de microservices ?* → un monolithe bien testé est plus convaincant qu'une architecture distribuée inachevée.
- *Comment évites-tu la survente sur un cycle ?* → mise à jour atomique (`findOneAndUpdate` avec condition sur la quantité disponible) + transaction MongoDB multi-documents dans `reservation`.
- *Pourquoi un logo abstrait sans animal ?* → plateforme multi-espèces (voir document de proposition, "Nom de la plateforme").
- *Qu'est-ce qui est réellement anti-scam ici, concrètement ?* → vérification d'identité, score de confiance calculé, paiement séquestre, rate limiting.

### 5. Checklist finale avant de publier le portfolio

- [ ] Dépôt public, README à jour (commandes vérifiées une dernière fois de zéro, sur une machine propre)
- [ ] CI verte sur le dernier commit
- [ ] Licence présente
- [ ] Captures d'écran ou courte vidéo de démo ajoutées au README
- [ ] Lien du projet ajouté au CV / LinkedIn avec une phrase qui reprend le "pourquoi" du projet, pas juste la stack

## Feuille de route post-MVP (vers un produit complet)

Les phases 1 à 6 (ci-dessus) produisent un portfolio fini et démontrable. Au-delà, voici ce qu'il faudrait construire pour transformer Betailo en produit réellement exploitable.

| Phase | Thème | Contenu clé | Ce que ça ajoute |
| --- | --- | --- | --- |
| 7 | Paiements & logistique réels | Intégration Mobile Money (Orange Money, MTN MoMo) et carte via un prestataire (Stripe ou équivalent local), frais et suivi de livraison, factures/reçus automatiques | Passe du paiement simulé à de l'argent réel — implique conformité et gestion des échecs de paiement |
| 8 | Zootechnie avancée | Rappels de vaccination programmés, fiches techniques enrichies par des données vétérinaires de référence, traçabilité par QR code sur chaque lot | Différenciation métier plus forte face aux marketplaces génériques |
| 9 | Communauté enrichie | Messagerie privée éleveur ↔ acheteur, réputation croisée forum + transactions, groupes régionaux et annonces d'événements (foires, marchés) | Renforce la rétention et l'effet réseau |
| 10 | Anti-fraude poussé | Vérification d'identité automatisée (KYC via un prestataire type Smile Identity), détection de comportements suspects par modèle statistique plutôt que règles fixes, processus de litige/médiation structuré | Passe d'un anti-scam "raisonnable pour un MVP" à une vraie ligne de défense produit |
| 11 | Scalabilité & performance | Cache (Redis) sur les listings/recherche, recherche full-text et géolocalisée (index MongoDB `$text` et `2dsphere`), lecture sur les répliques secondaires du cluster Atlas | Nécessaire seulement si le trafic réel le justifie — pas à construire par anticipation |
| 12 | Observabilité & fiabilité | Tracing distribué (OpenTelemetry), dashboards Grafana/Prometheus, alerting, sauvegardes automatisées et plan de reprise | Indispensable dès qu'il y a de vrais utilisateurs et de l'argent en jeu |
| 13 | Mobile & accès en zone rurale | Application mobile ou PWA, mode hors-ligne partiel (cache local + synchronisation différée), fonctions clés accessibles par SMS/USSD | Répond à une connectivité souvent instable chez le public cible |
| 14 | Internationalisation | Multi-devises, multi-langues (au-delà du français), adaptation réglementaire par pays en cas d'expansion régionale (zone CEMAC, Afrique de l'Ouest) | Pertinent seulement si une expansion géographique est envisagée |
| 15 | Monétisation | Commission sur transactions, abonnement premium éleveur (mise en avant d'annonces, statistiques avancées), programme de fidélité | Transforme le projet en business model viable |
| 16 | Conformité & gouvernance des données | Politique de confidentialité et CGU/CGV conformes à la réglementation locale (cybersécurité/cybercriminalité au Cameroun), RGPD si utilisateurs européens | à traiter avant toute ouverture publique réelle, pas après |

### Comment prioriser au-delà du MVP

L'ordre ci-dessus n'est pas strict : en usage réel (par opposition au portfolio), l'ordre logique serait plutôt 7 (paiements réels) → 16 (conformité, à traiter en même temps que 7, pas après) → 10 (anti-fraude poussé, critique dès que de l'argent réel circule) → 13 (mobile/hors-ligne, condition d'adoption réelle par les éleveurs) → puis 8, 9, 11, 12, 14, 15 selon la traction observée. Les phases 11 (scalabilité) et 14 (internationalisation) en particulier ne devraient être lancées que lorsque l'usage réel le justifie — les construire trop tôt est un piège classique ("premature scaling") qui coûte du temps sans bénéfice immédiat.
