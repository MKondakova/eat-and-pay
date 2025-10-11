# Backend для мобильного приложения

Доставка еды и кошелек

## 📘 API

Полное описание всех методов доступно в OpenAPI [спецификации](api/openapi/spec.yaml).

## 🚀 Установка и запуск

Для работы требуется установленный **nginx** и **Docker**.

1. Настроить nginx:

   ```shell
   cp eats-pages.ddns.net.conf /etc/nginx/sites-available/eats-pages.ddns.net.conf
    
   sudo ln -s /etc/nginx/sites-available/eats-pages.ddns.net.conf /etc/nginx/sites-enabled/eats-pages.ddns.net.conf
    
   sudo nginx -t
   sudo nginx -s reload
   ```

2. Собрать и запустить контейнер (приложение работает на порту `8080` внутри контейнера):

   ```shell
   docker build . -t eats-pages-image
    
   docker rm -f eats-pages-app 

   docker run --env-file ./.env \
      -v "data:/root/data" \
      --restart always \
      -p 8081:8080 \
      -d --name eats-pages-app eats-pages-image:latest
   ```

---

## 📊 Структура данных

Приложение загружает начальные данные из JSON файлов в папке `data/`.

### Файлы данных

#### products.json
Содержит массив товаров. Каждый товар имеет следующие поля:
- `id` - уникальный идентификатор товара
- `image` - URL изображения товара
- `name` - название товара
- `weight` - вес в граммах
- `price` - цена в копейках
- `rating` - рейтинг товара (0-10)
- `description` - описание товара
- `discount` - размер скидки в процентах
- `reviews` - массив отзывов
- `available` - доступность товара

#### categories.json
Содержит массив категорий. Каждая категория имеет:
- `id` - уникальный идентификатор категории
- `name` - название категории
- `image` - URL изображения категории

#### product_categories.json
Содержит связки товаров и категорий в формате:
```json
{
  "category_id": ["product_id1", "product_id2"]
}
```

#### blocked_tokens.json
Содержит массив заблокированных JWT токенов.

#### created_tokens.json
Содержит массив созданных JWT токенов для отслеживания.

### Архитектура загрузки данных

```
JSON файлы (data/) 
    ↓
Config (internal/config/config.go)
    ↓ загружает и парсит JSON
Application (internal/application/application.go)
    ↓ передает данные из config
Services (internal/service/)
    ↓ получает данные для работы
```

### Загрузка данных

Данные загружаются автоматически при запуске приложения через config. Если файлы не найдены или содержат ошибки, приложение продолжит работу с пустыми данными и выведет предупреждение в логах.

### Расширение данных

Для добавления новых товаров или категорий просто отредактируйте соответствующие JSON файлы. Приложение автоматически подхватит изменения при следующем запуске.
