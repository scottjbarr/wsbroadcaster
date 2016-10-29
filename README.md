# Web Socket Broadcaster

The Web Socket Broadcaster (wsbroadcaster) is a generic middleware service
that provides a broadcast web socket endpoint.

Data comes in via a Redis key, and is broadcast out to all listening clients.

There is no http handler, this is just the websocket handler and upgrade
itself.


## Usage

    BIND=:10000 REDIS_URL=redis://localhost:6379 wsbroadcaster


## Licence

The MIT License (MIT)

Copyright (c) 2016 Scott Barr

See [LICENSE.md](LICENSE.md)
