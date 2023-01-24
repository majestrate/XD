# Pour débuter

Une fois que vous avez compilé ou obtenu une version, une interface web est
disponible par défaut à http://127.0.0.1:1488/.

Les utilisateurs de Windows sont encouragés à utiliser l'interface web s'ils ne
peuvent pas utiliser l'outil en ligne de commande.

## Ligne de commande (Unix)

Pour utiliser l'outil en ligne de commande, vous devez établir un lien
symbolique de `XD` vers `XD-cli`:

    $ ln -s XD XD-cli

Pour ajouter des torrents depuis un fichier de semence à travers i2p:

    XD-cli add http://somesite.i2p/some/url/to/a/torrent.torrent

Pour lister les torrents actifs:

    XD-cli list

Pour augmenter le nombre de morceaux à demander en parallèle, utilisez la
commande `set-piece-window` (celle-ci sera peut-être retirée dans le futur):

    XD-cli set-piece-window 10

## Ligne de commande (Windows)

Sous Windows, faites une copie de `XD.exe` et appelez-la `XD-cli.exe`.

Toutes les commandes sont comme sous Unix, mis à part que `/` peut avoir besoin
d'être échappé, dépendant du terminal utilisé.

TODO: ajouter plus de documentation pour Windows

# Configuration

XD utilise le format ini pour sa configuration. Le fichier de configuration
principal est `torrents.ini` et est autogénéré avec les valeurs par défaut s'il
n'est pas présent.

## Configuration pour stockage SFTP

XD peut utiliser un système de fichier distant accédé par SFTP. Pour utiliser
cette fonctionnalité, elle doit être configurée.

Exemple de configuration:

    [storage]
    rootdir=/mnt/storage/XD/
    metadata=/mnt/storage/XD/metadata
    downloads=/mnt/storage/XD/downloads
    sftp_host=remote.server.tld
    sftp_port=22
    sftp_user=votre_usager_ssh
    sftp_remotekey=cle_publique_du_serveur_en_base64
    sftp_keyfile=/chemin/vers/cle/ssh/privee/id_rsa
    sftp=1

Cet exemple établit une connection à `remote.server.tld:22` avec l'usager
`votre_usager_ssh` en utilisant la clé privée (non-chiffrée) et utilise
`/mnt/storage/XD/` sur le serveur distant pour stocker les torrents et les
méta-données.

La clé publique du serveur est habituellement située dans
`/etc/ssh/ssh_host_*.pub` sous la forme
`ssh-quelque-chose la_cle_est_ecrite_ici_en_base64 root@hostname`.
La valeur en base64 doit être utilisée pour `sftp_remotekey`.

