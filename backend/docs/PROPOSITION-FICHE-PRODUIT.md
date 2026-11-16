# Fiche produit et CRUD Produits : ce que prévoit le contrat, avant tout code

Statut : **PROPOSITION, en attente de vos corrections.** Aucun code n'est écrit. Source : `api/openapi.yaml` (tags « Admin · Catalogue » : **brouillon**, jamais validé) et la section 3 de votre « Spécification complète du backoffice ».

## 1. La fiche produit, telle que le contrat la prévoit

| Champ | Type / règle | Obligatoire |
|---|---|---|
| `sku` | texte, 64 caractères maximum | oui |
| `slug` | texte ou vide (adresse lisible) | non |
| `name` | texte **par langue** : `{"fr": "…", "en": "…"}`, au moins la langue par défaut | oui |
| `description` | texte par langue | non |
| `brandId` | marque (vide possible) | non |
| `categoryId` | catégorie (une seule) | oui |
| `status` | `draft` (brouillon) \| `active` \| `archived` | oui |
| `fulfillmentMode` | `stock` \| `made_to_order` (commandable sans stock) | oui |
| `leadTimeDays` | entier ≥ 0, **obligatoire si `made_to_order`** | conditionnel |
| `attributes` | liste `{clé, valeur}` ; la valeur est du texte, un nombre ou un booléen selon la **définition de l'attribut de la catégorie** | non |
| `sectorIds` | secteurs d'activité rattachés | non |

**Rattachés à la fiche, avec leurs propres routes :**
- **Médias** : URL, texte alternatif, ordre (ajout, réordonnancement, suppression).
- **Compatibilités** : machines compatibles + note.
- **Références croisées** : marque, numéro, type `oem` \| `equivalent` \| `supersedes`.
- **Prix** : **hors taxes**, en XOF (devise de base), prix fixe par devise ou **dérivé** du XOF par un taux enregistré ; paliers de remise par quantité.
- **Disponibilité** (lecture seule, calculée depuis le stock) : `in_stock` \| `low_stock` \| `out_of_stock` \| `made_to_order`, délai d'expédition.

## 2. Ce qui doit exister avant le premier produit

| Entité | Champs du contrat |
|---|---|
| **Catégorie** | `slug` (minuscules et tirets), parent (arbre), ordre, nom et description par langue ; déplaçable dans l'arbre |
| **Définition d'attribut** (par catégorie, **héritée** par les sous-catégories) | clé (`a-z`, chiffres, `_`), libellé par langue, type `text` \| `number` \| `boolean` \| `enum`, unité (mm, µm, bar, kW…), **filtrable**, **obligatoire**, options (pour `enum`), ordre |
| **Marque** | `slug`, nom, logo (URL) ; compteur de produits (calculé) |
| **Secteur** | `slug`, nom, image (URL) |

## 3. Routes de la phase « ajout manuel »

`GET/POST /admin/products`, `GET/PUT/DELETE /admin/products/{id}`, médias (3 routes), compatibilités (1), prix par devise (2), import et export de produits (2) ; catégories (6), marques (5), définitions d'attributs (5).

## 4. Écarts entre le contrat et votre spécification du backoffice

| Votre spécification | Le contrat actuel | Écart |
|---|---|---|
| Statuts **brouillon, à valider, publié** | `draft`, `active`, `archived` | pas d'étape « à valider » |
| **Historique des versions** et **origine de chaque champ** | absents | à créer |
| **Dupliquer** un produit | absent | à créer |
| **Score de complétude** par fiche | absent | à créer |
| **Conversions d'unités** (mm / pouces) | une unité par attribut, sans conversion | à créer |
| **Fournisseurs** et **offres multiples** par produit (prix, délai, stock, lien source) | **un seul prix** par produit (vente directe) ; aucun fournisseur | modèle différent : voir question 9 |
| **Documents techniques** (fiches, manuels, certificats) | absents | à créer |
| **Import CSV/Excel avec correspondance des colonnes et rapport d'erreurs ligne par ligne** | une route d'import et une d'export, sans détail | à préciser |
| **Qualité des données**, **dédoublonnage** | absents | phase 2 du moteur |
| Contenus multilingues, montants multidevises | pris en charge (`LocalizedText`, prix par devise) | conforme |

## 5. Mes recommandations et les questions à trancher

1. **Statuts.** Ajouter `review` (« à valider ») entre `draft` et `active` : brouillon → à valider → publié → archivé. Seul un membre ayant la permission de publication passe une fiche en `active`. *À valider.*
2. **SKU.** Unique (insensible à la casse), 64 caractères, sans espace en début ou fin, **non modifiable une fois le produit publié** (les commandes et les devis y renvoient). Format imposé (ex. majuscules, chiffres, `-`) : **[À REMPLIR — vous]**.
3. **Langue par défaut obligatoire** : `fr` ? Les autres langues sont facultatives avec repli sur la langue par défaut (le contrat prévoit déjà ce repli). **[À confirmer]**
4. **Publication.** Une fiche ne peut passer `active` que si : nom dans la langue par défaut, catégorie, prix dans la devise de base **(si vente directe)**, et **tous les attributs obligatoires** de sa catégorie renseignés. Sinon refus avec la liste des manques. *Je recommande ce blocage.*
5. **Suppression.** `DELETE` = **archivage** si le produit a déjà servi (commande, devis, panier) ; suppression réelle uniquement pour un brouillon jamais utilisé. Cohérent avec la règle « désactiver, jamais supprimer ». *À valider.*
6. **Slug** : calculé automatiquement depuis le nom (par langue), modifiable avant publication, **figé après** (liens et référencement).
7. **Historique et origine des champs** : une table de versions (chaque enregistrement conserve l'état avant/après, l'auteur, la date) ; l'**origine** d'un champ prend les valeurs `manuel` ou `import` pour l'instant ; les sources du moteur viendront en phase 2. *Je recommande de les inclure dès le départ : l'ajouter après coûte une migration de toutes les fiches.*
8. **Médias.** Le contrat n'accepte qu'une **URL**. Téléverser des fichiers exige un stockage d'objets (où ? **[À REMPLIR — vous]**). Proposition pour démarrer : **URL seulement**, téléversement dans un second temps.
9. **Offres de fournisseurs externes vs vente directe sur la même fiche** (question centrale du modèle « les deux », déjà annoncée) : pour l'ajout manuel, **une fiche = un prix de vente direct**. Les offres externes (lien de renvoi, prix indicatif, date de vérification) arrivent en phase 2 avec « Fournisseurs » et « Offres et prix », sans casser la fiche. *À valider.*
10. **Prix et TVA.** Prix **hors taxes**, en XOF. Le taux de TVA applicable n'est **pas encore défini** (page Taxes, phase 6) : la fiche n'affiche donc **que du hors taxes**, jamais de toutes taxes comprises, tant que ce n'est pas renseigné. *À confirmer.*
11. **Identifiants complémentaires** absents du contrat : code-barres GTIN/EAN, **unité de vente** (pièce, lot de N), poids et dimensions pour la livraison. Utiles ? **[À décider — vous]** : je ne les ajoute pas sans votre accord.
12. **Stock.** Hors de cette étape (onglet Stock, phase 4) : un produit `stock` s'affiche « rupture » tant qu'aucun stock n'est saisi ; `made_to_order` reste commandable. *À confirmer.*
13. **Import CSV.** Première version : colonnes fixes (celles de la fiche), rapport d'erreurs ligne par ligne, **tout ou rien par ligne** (une ligne en erreur n'empêche pas les autres), modèle de fichier téléchargeable. La correspondance de colonnes libre vient ensuite.

## 6. Ordre de réalisation proposé (après vos corrections)

1. Catégories (arbre) → définitions d'attributs → marques → secteurs : contrat validé, service `catalog`, base, écrans.
2. Produits : fiche, médias par URL, références croisées, compatibilités, prix en XOF, versions et audit, publication contrôlée.
3. Import/export CSV.
4. Branchement du site : les pages catalogue, produit et panier lisent ces données (fin de l'état vide).
5. Tests : intégration PostgreSQL, contraintes en base (SKU unique, attributs typés, `made_to_order` sans délai refusé), concurrence, audit, permissions.

Permissions existantes : `catalog:read`, `catalog:write` ; je propose d'y ajouter `catalog:publish` (publier une fiche), comme `content:publish`. *À valider.*

---

## 7. Décisions du 17 nov. 2026 : fiche produit VALIDÉE avec corrections

**Ajouts à la fiche**
- `manufacturerRef` : référence constructeur (normalisée ; unique par marque).
- `priceMode` : `fixed` | `on_quote` (sur devis).
- Champs facultatifs dès maintenant : GTIN/EAN, unité de vente, quantité minimale de commande, poids, dimensions.

**Réponses aux 13 points**
1. Statut `review` conservé ; un membre ayant `catalog:publish` peut passer directement de brouillon à publié.
2. SKU : majuscules, chiffres et tirets ; généré automatiquement si vide ; figé après publication.
3. Langue par défaut `fr`, avec repli.
4. Blocage à la publication si des champs obligatoires manquent, en tenant compte de `priceMode`.
5. Suppression = archivage si le produit a servi.
6. Slug figé après publication.
7. Historique des versions et origine des champs (`manual` / `import`) dès le départ.
8. Médias par URL au départ ; le **téléversement (stockage compatible S3) doit être livré avant l'ouverture au public** et servira aussi aux documents techniques.
9. Une fiche = un prix de vente direct ; offres externes en phase 2.
10. Prix hors taxes uniquement. **Devise de base : à confirmer (XOF ou EUR)** ; réglage `catalog_settings.base_currency`, XOF par défaut provisoire.
11. Voir « Ajouts à la fiche ».
12. Tant que l'onglet Stock n'existe pas : « Disponibilité sur demande », jamais « rupture ».
13. Import CSV à colonnes fixes avec mode simulation.

Permission `catalog:publish` créée. Ordre de réalisation validé : catégories → attributs → marques → secteurs → produits → import/export CSV → site branché → tests ; puis service `search` ; puis écrans du moteur.

**Avancement** : l'étape « taxonomie » (catégories, attributs, marques, secteurs) est livrée (service `catalog`, routes du gateway, écrans du backoffice, audit). Un attribut ne change ni de clé, ni de type, ni de catégorie une fois créé.
