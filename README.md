# Dice Server

Server for the dice games, with currently implemented game of farkle.

The game can be played at [dice.jinderamarak.cz](https://dice.jinderamarak.cz).

## Architecture
```
                                                   +--------------+       +-----------------+
                                                   |              | <---> |   Game Server   | <---[ WebSocket ]--->   Players
                         +-----------------+       |              |       +-----------------+
Players <---[ HTTP ]---> |   Game Master   | <---> |   RabbitMQ   |     
                         +-----------------+       |              |       +-----------------+
                                                   |              | <---> |   Game Server   | <---[ WebSocket ]--->   Players
                                                   +--------------+       +-----------------+
```