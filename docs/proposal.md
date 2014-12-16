
Config Store
============

The config store is a fuse mounted file system backed against a distributed key/value store; i.e. think etcd but accessable in a file system. The file system will also have native support for templated configuration files similar to confd, which can pull from both the KV store and well as service discovery backends such as Consul.