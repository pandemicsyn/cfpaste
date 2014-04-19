cfpaste
=======

Go based Openstack Swift/Rackspace CloudFiles pastebin

Demo
====

http://paste.ronin.io/


Installation
============

1. go get github.com/pandemicsyn/cfpaste
2. go install github.com/pandemicsyn/cfpaste
3. cd $GOPATH/src/github.com/pandemicsyn/cfpaste
5. edit and source dot-swift-creds
6. run $GOPATH/bin/cfpaste
7. profit!

Make it faster
==============

1. Fire up memcached on 127.0.0.1:11211