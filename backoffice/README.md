# FilterFinder — Backoffice

Interface d'administration (Next.js) — **non démarré**. Dossier créé en attente du brief (écrans, rôles, permissions).

Dépendances côté backend : authentification du personnel (2FA, invitations) dans `backend/services/identity`, routes `/admin/*` du contrat `backend/api/openapi.yaml`.

## Démarrage

```
cp .env.example .env.local
npm run dev   # http://localhost:3000
```

Squelette Next.js (App Router, TypeScript, Tailwind) : `/login` (formulaire non branché) et `/dashboard` (coquille). Le reste attend le brief.

## Écrans

Connexion + 2FA, invitations, **personnel**, **rôles et permissions**, **journal d'audit**, **mon compte**, **clients** (liste filtrée, détail, suspension/réactivation) et **entreprises** (file « en attente », approbation, rejet avec motif). Le tableau de bord affiche le nombre d'entreprises à valider.

Pas encore : demandes de données personnelles (`/admin/data-requests`, 501 : le stockage des demandes n'existe pas), nombre de commandes et valeur vie client d'un client (service des commandes absent : 0 / null).
