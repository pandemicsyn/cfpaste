cfpaste
=======

Go based Openstack Swift/Rackspace CloudFiles pastebin

Demo
====

http://paste.ronin.io/c7dcb578

Installation
============

1. go get github.com/pandemicsyn/cfpaste (also go get -u github.com/ncw/swift if you get a 400 on auth)
2. go install github.com/pandemicsyn/cfpaste
3. cd $GOPATH/src/github.com/pandemicsyn/cfpaste
5. edit and source dot-swift-creds
6. run $GOPATH/bin/cfpaste
7. profit!

Make it faster
==============

1. Fire up memcached on 127.0.0.1:11211
