# apt-repo-queue 


apt-repo-queue is a tool to manage the other end of a dput/dupload, so
your developer can upload package to an archive. It is suited for
small to medium archive, where several people should be authorized to
upload to the archive.

It features :
   
* Multiple distribution support
* Multiple component support
* One package can only be upload to one distribution (reprepro limitation)


## Workflow

A directory is watched using filesystem notification for incoming
,changes file. Then the PGP signature of these file are verified
against a private keyring of authorized keys. Then reprepro ios
managing a small repository.

## Installation and initialization

```bash
go get ponyo.epfl.ch/gitlab/alexandre.tuleu/go-deb/apt-repo-queue
sudo install -m 755 $(which apt-repo-queue) /usr/local/bin
apt-queue-repo init
```

## Seting up a daemon


