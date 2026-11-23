# internal/config

Пакет загружает настройки HTTP-сервера через `config.Parse()`.

| Поле      | Переменная окружения | Флаг | Значение по умолчанию   |
|-----------|----------------------|------|-------------------------|
| `RunAddr` | `SERVER_ADDRESS`     | `-a` | `localhost:8080`        |
| `BaseURL` | `BASE_URL`           | `-b` | `http://localhost:8080` |

Для каждого поля независимо действует приоритет: непустая переменная окружения → флаг → значение
по умолчанию. Пустая переменная окружения не переопределяет значение, полученное при разборе флагов.
`RunAddr` определяет адрес прослушивания, а `BaseURL` — базовый адрес возвращаемых коротких ссылок.

Запускайте из корня проекта (bash/zsh):

```sh
SERVER_ADDRESS=localhost:8081 BASE_URL=http://localhost:8081 go run ./cmd/shortener
```

При разных значениях окружение имеет приоритет над флагами:

```sh
SERVER_ADDRESS=localhost:8082 BASE_URL=https://short.example go run ./cmd/shortener -a localhost:8081 -b http://localhost:8081
```

Здесь сервер слушает `localhost:8082`, а короткие ссылки начинаются с `https://short.example/`.
