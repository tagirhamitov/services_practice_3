# Практика 2: Мафия

## Реализована функциональность на 10 баллов

Код написан на языке Go.

Код клиента в директории `mafia_client`. Внутри есть директории `manual` и `automatic` - для ручного клиента и бота.

Код сервера мафии в директории `mafia_server`.

Код чат сервера в директории `chat_server`.

Proto-схема в директории `proto`.

У чат сервера есть выделенная очередь `events`, через которую создаются и удаляются чат-комнаты. Также у каждой сессии есть очередь с именем равным ID сессии. В нее отправляются сообщения. Также у каждого пользователя есть своя очередь с именем равным имени пользователя. В нее перенаправляются сообщения от чат сервера.

## Запуск
Следующая команда запускает сервер и двух ботов через `docker-compose`:
```shell
$ docker-compose up --build
```
Количество ботов можно изменить, модифицировав файл `docker-compose.yml`

Чтобы открыть ручных клиентов, нужно собрать образ и запустить:
```shell
$ docker build . -f mafia_client.dockerfile -t mafia_client
$ docker run -it --net=host mafia_client
```

После этого клиен попросит ввести имя пользователя и адрес сервера (по умолчанию нужно ввести `:9000`).

Альтернативно, остальных клиентов можно также запустить в автоматическом режиме:
```shell
$ docker run --net=host mafia_client --auto --username USERNAME --server SERVER
```
где вместо `USERNAME` нужно подставить любое имя пользователя, а вместо `SERVER` - адрес сервера (по умолчанию нужно ввести `:9000`).

## Использование

В дополнение к [предыдущему функционалу](https://github.com/tagirhamitov/services_practice_2/blob/main/README.md) в новой версии клиента появилась команда:
- `msg <MESSAGE>`

Вместо `<MESSAGE>` может быть любая строка с сообщением. Данное сообщение будет отправлено остальным игрокам в текущей сессии и выведено в их поток вывода.

## Чат в ночное время
В код сервера мафии добавлен rpc метод `ActivePlayers`. Он возвращает список игроков, которые могут общаться на текущем этапе игры. Днем это все живые пользователи, ночью - только мафии.

В файле [chat.go](chat_server/chat/chat.go) вызывается этот метод и проверяется, что отправитель может общаться, а также сообщение перенаправляется только тем игрокам, которые также могут общаться.