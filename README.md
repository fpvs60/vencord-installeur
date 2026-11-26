# Installeur Vencord

Installe Vencord dans Discord avec ma sélection de plugins (et leurs réglages) et le thème NotAnotherAnimeTheme.

## Installer

Ouvre **PowerShell** (touche Windows, tape `powershell`, Entrée), colle cette ligne et appuie sur Entrée :

```powershell
irm https://raw.githubusercontent.com/fpvs60/vencord-installeur/main/install.ps1 | iex
```

Choisis `1` pour installer. Discord se ferme, s'installe et se relance tout seul.
Pour retirer Vencord, relance la même commande et choisis `2`.

Ce que fait l'installeur :

- sauvegarde ton Vencord et tes réglages actuels (`%APPDATA%\Vencord\*.bak-…`) ;
- installe le build Vencord, le thème et active les plugins, en gardant tes autres réglages ;
- injecte Vencord dans Discord, PTB et Canary (l'original est gardé en `_app.asar`).

## Pour moi : mettre à jour

```powershell
.\build.ps1 -Publish
```

Reprend mon build Vencord, mes plugins et mon thème actuels, recompile l'exe et publie une nouvelle release.

## Licence

Vencord est sous [GPL-3.0](https://github.com/Vendicated/Vencord/blob/main/LICENSE). Le build inclus est Vencord
([Vendicated/Vencord](https://github.com/Vendicated/Vencord)) avec la modification de `patches/permissionsviewer.patch`.
