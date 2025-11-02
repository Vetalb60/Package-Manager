General Information
-------------------

Package manager is a cli util for CRUD operations with remote file server.

-------------------

### Package Manager CLI

    Remote client for remote storage
    
    Usage:
        RemoteClient [command]
    
    Available Commands:
        completion  Generate the autocompletion script for the specified shell
        create      create a new package
        fetch       download exist package from storage
        help        Help about any command
        remove      remove exist package
        update      create a new or update existing package
    
    Flags:
        -f, --cfg string            configs file (default is empty)
        -e, --env                   read configs from environment
        -h, --help                  help for RemoteClient
        -o, --output string         path for save fetching packages (default ".")
        -p, --pack string           input packet.json (default "packet.json")
        -s, --storage_path string   path in remote storage server for saving files (default ".")
        -u, --unpack string         input packages.json (default "packet.json")
    
    Use "RemoteClient [command] --help" for more information about a command.

-------------------

### QuickStart:

**Configs**: _.env file_
	
	UPLOADER_SSH_PASSWORD="password"
    UPLOADER_SSH_USERNAME="user"
    UPLOADER_SSH_HOST="localhost"
    UPLOADER_SSH_PORT=22
    UPLOADER_SSH_TIMEOUT=5
    UPLOADER_SSH_PRIVATE_FILE="<path to id_rsa>"
    UPLOADER_SSH_STORAGE_PATH="<package storage location>"
    UPLOADER_KEY_EXCHANGES="diffie-hellman-group-exchange-sha256;diffie-hellman-group14-sha256"
_.json file_

    {
        "ssh": {
            "username": "user",
            "password": "password",
            "timeout": 5,
            "host": "localhost",
            "port": 22,
            "private-key": "",
            "ssh-key-exchanges": [
                "diffie-hellman-group14-sha256",
                "diffie-hellman-group-exchange-sha256"
            ],
            "ssh-storage-path": "remote"
        }
    }

tests:

    go test ./...

build:

    go  build -o rc main.go

operations:

    ./rc create -f ./configs/.remote.uploader.json -p packet.json -o remote
    ./rc update -f ./configs/.remote.uploader.json -p packet.json -o remote
    ./rc fetch -f ./configs/.remote.uploader.json -u packages.json -o remote
    ./rc remove -f ./configs/.remote.uploader.json -u packages.json -o remote

-------------------

G-mail: **al9xgr99n@gmail.com**