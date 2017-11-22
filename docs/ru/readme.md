# Начало работы

После того как вы собрали и запустили клиент, веб-интерфейс будет доступен по адресу: http://127.0.0.1:1488/
Пользователям Windows рекомендуется использовать веб-интерфейс, если они не используют командную строку

## Командная строка (Unix)

Чтобы использовать командную строку, вы должны создать символьную ссылку к `XD-cli`:

    $ ln -s XD XD-cli

Добавить торрент через i2p:

    $ XD-cli add http://somesite.i2p/some/url/to/a/torrent.torrent

Список активных торрентов:

    $ XD-cli list

Увеличить количество параллельных запросов (может быть удалено в будущем):

    $ XD-cli set-piece-window 10

## Командная строка (Windows)

В Windows: сделайте копию файла с именем `XD-cli.exe`
Все команды выполняются таким же образом, как и в `Unix`, но зависят от используемого терминала.

## Конфигурация

XD использует формат файла ini для конфигурации, основной файл конфигурации называется torrents.ini и автогенерируется со значениями по умолчанию.

## Конфигурация хранилища SFTP

XD может использовать удаленную файловую систему, доступную через `sftp`.

### Пример настройки:

    [storage]
    rootdir=/mnt/storage/XD/
    metadata=/mnt/storage/XD/metadata
    downloads=/mnt/storage/XD/downloads
    sftp_host=remote.server.tld
    sftp_port=22
    sftp_user=your_ssh_user
    sftp_remotekey=base64dserverpublickeygoeshere
    sftp_keyfile=/path/to/ssh/private/key/to/login/with/id_rsa
    sftp=1

Это позволит подключиться к `remote.server.tld:22` пользователю `your_ssh_user` с помощью (незашифрованного) закрытого ключа и использовать `/mnt/storage/XD/` на удаленном сервере в качестве хранилища для торрентов и метаданных.

Открытый ключ сервера обычно находится `/etc/ssh/ssh_host_*.pub` в формате: `ssh-whatever base64goeshere root@hostname` вы будете использовать значение base64 в `sftp_remotekey`.
