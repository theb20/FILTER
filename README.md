# FilterFinder

![FilterFinder Logo](./src/assets/logo.png)  

**FilterFinder** est une application web permettant aux utilisateurs de **rechercher et comparer des produits ou pièces automobiles**, passer des commandes (devis), consulter des annuaires intelligents et filtrer selon plusieurs critères (prix, catégorie, marque, etc.).

---

## 🚀 Fonctionnalités

- Recherche intelligente de produits/pièces automobiles  
- Filtrage par catégorie, prix, marque, etc.  
- Comparaison de produits  
- Gestion des devis et commandes  
- Mode sombre / clair  
- Interface responsive adaptée aux mobiles et tablettes  

---

## ▶️ Lancer tout l'environnement local (site + backoffice + backend)

```
./dev.sh up                                   # PostgreSQL, NATS, identity, gateway, site, backoffice
./dev.sh admin votre@email.com "Votre Nom"    # une seule fois : crée le premier super-admin (mot de passe saisi au terminal)
./dev.sh status | logs | down
```

| Brique | Adresse |
|---|---|
| Site (Vite) | http://localhost:5173 |
| Backoffice (Next.js) | http://localhost:3100/login |
| API (gateway) | http://127.0.0.1:8080 (`/healthz`) |

Le site appelle l'API par un relais Vite (`/v1`) : même origine, cookie de session `HttpOnly` + `SameSite=Strict`. Le backoffice l'appelle depuis son serveur. Les données sont dans `~/.filterfinder` (conservées par `./dev.sh down`). Détails : `backend/README.md`.

---

## 🏗️ Architecture du projet

FilterFinder utilise une architecture **3 tiers** : Frontend, Backend, Base de données.

   +------------------+
   |   Frontend       |  <- React + Tailwind
   +------------------+
            |
            v
   +------------------+
   |   Backend        |  <- Node.js + Express
   +------------------+
            |
            v
   +------------------+
   | Base de données  |  <- MySQL / PostgreSQL
   +------------------+
# ⚠️ Avertissement de licence

© 2025 HL. Tous droits réservés.

Ce projet et tout son contenu (code source, images, textes, designs, logos, données, etc.) sont la **propriété exclusive de FilterFinder**.  

⚠️ **Conditions d’utilisation :**
- Il est **strictement interdit** de copier, modifier, distribuer ou utiliser ce projet sans l’autorisation écrite préalable de FilterFinder.
- Toute utilisation non autorisée pourra entraîner des **poursuites légales** conformément aux lois sur la propriété intellectuelle.

Pour obtenir une autorisation ou une licence d’utilisation, contactez : **contact@filterfinder.com**.
