`
# chili-traducteur-go 🌶️

chili-translator-go est un wrapper de traduction automatique universel écrit en Go. Il est conçu pour traduire des scripts (.sh, .py), des fichiers de documentation (Markdown) et des fichiers de données (JSON) tout en préservant l'intégrité des variables, des liens et de la syntaxe technique.

Son principal avantage est Smart Cache v2.1.9, qui réduit considérablement les appels réseau et accélère les traductions répétitives grâce à la réutilisation locale des données.

## ✨ Caractéristiques

* Multiformat : prend en charge .sh, .py, .md, .json, .yaml.
* Préservation de la syntaxe : protège automatiquement les variables shell ($VAR, ${VAR}), les liens Markdown et les espaces réservés de chaîne pendant le processus de traduction.
* Traduction parallèle : traite plusieurs langues simultanément à l'aide de Goroutines (réglable via -j).
* Cache persistant avec horodatage : stocke les traductions localement et gère le cycle de vie des données, permettant un nettoyage intelligent.
* Interface progressive : affichage en temps réel de la progression de chaque langue avec un alignement visuel parfait, quelle que soit la taille du code de langue (par exemple en vs zh-CN).

## 🚀Installation

Assurez-vous que Go est installé et que les dépendances du système (gettext, trans).
```bash
git clone https://github.com/chililinux/chili-tradutor-go.git
cd chili-traducteur-go/src
allez construire -o chili-translator-go chili-translator-go-v2.1.9.go
sudo mv chili-translator-go /usr/local/bin/
```

## 🛠️ Utilisation

### Traduction de base
Pour traduire un fichier dans des langues standards (EN, ES, IT, DE, FR, RU, ZH, JA, KO) :

chili-translator-go -i meu_script.sh


### Spécification des langues et du moteur

cheli-treducer-go -et tutoriel.md


### Effacement du cache
Supprimez les entrées de cache qui n'ont pas été utilisées au cours des 30 derniers jours :

chili-translator-go --clean-cache


## ⚙️ Options (Drapeaux)

| Drapeau | Longue | Descriptif |
| :--- | :--- | :--- |
| -je | --fichier d'entrée | Fichier source pour la traduction. |
| -e | --moteur | Moteur de traduction : Google, Bing, Yandex (par défaut : Google). |
| -s | --source | Langue source (ex : pt, en) (par défaut : auto). |
| -l | --langue | Liste des langues séparées par une virgule ou toutes. |
| -j | --emplois | Nombre de traductions simultanées (par défaut : 8). |
| -f | --force | Force la traduction en contournant le cache local. |
| | --clean-cache | Supprime les éléments du cache obsolètes (> 30 jours). |
| -q | --calme | Mode silencieux (pas de progression visuelle). |
| -v | --verbeux | Affiche les détails techniques pendant l'exécution. |
| -V | --version | Affiche la version actuelle. |

## 📁 Structure de sortie

* Scripts/POT : génère des fichiers .po dans ./pot/ et des fichiers binaires .mo dans ./usr/share/locale/.
* Markdown : génère des versions traduites dans ./doc/ (ex : README-en.md).
* JSON : génère des versions traduites dans ./translated/.

## 🛡️ Logique de cache (v2.1.9)

Le cache est stocké dans ~/.cache/chili-tradutor-go/cache.json.

* Migration automatique : lors de la détection des enregistrements des versions précédentes (v2.1.8), l'outil marque automatiquement l'horodatage actuel sur les enregistrements hérités pour éviter la perte de données historiques.
* Mise à jour automatique : chaque fois qu'un élément est trouvé dans le cache, son horodatage "Dernière utilisation" est mis à jour, le protégeant ainsi d'un futur nettoyage automatique.
* Sécurité : le nettoyage via --clean-cache supprime uniquement ce qui est réellement hors d'usage, garantissant ainsi une croissance saine de votre base de connaissances en matière de traduction.


Développé par : Vilmar Catafesta <vcatafesta@gmail.com>
Copyright © 2023-2026 Équipe ChiliLinux
