# Backend для мобильного приложения

Доставка еды и кошелек

## 📘 API

Полное описание всех методов доступно в OpenAPI [спецификации](api/openapi/spec.yaml).

## 🚀 Установка и запуск

Для работы требуется установленный **nginx** и **Docker**.

1. Настроить nginx:

   ```shell
   sudo cp eats-pages.ddns.net.conf /etc/nginx/sites-available/eats-pages.ddns.net.conf
    
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

   В env файле необходимо установить PUBLIC_KEY и PRIVATE_KEY. Можно сгенерировать ключи командой:
   ```shell
   openssl genrsa -out private.pem 2048
   openssl rsa -in private.pem -pubout -out public.pem
   ```

   Затем конвертировать ключи в формат base64:
   ```shell
   cat public.pem | base64 -w 0 > public.base64
   cat private.pem | base64 -w 0 > private.base64
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

#### user_profiles.json
Содержит профили пользователей в формате:
```json
{
  "user_id": {
    "phone": "номер телефона",
    "name": "имя пользователя",
    "birthday": "дата рождения (YYYY-MM-DD)",
    "imageUri": "URL изображения профиля"
  }
}
```

#### cart_items.json
Содержит корзины пользователей в формате:
```json
{
  "user_id": {
    "product_id": {
      "id": "идентификатор товара",
      "quantity": "количество товара"
    }
  }
}
```

#### user_favourites.json
Содержит избранные товары пользователей в формате:
```json
{
  "user_id": ["product_id1", "product_id2"]
}
```

#### orders.json
Содержит заказы пользователей в формате:
```json
{
  "user_id": [
    {
      "id": "идентификатор заказа",
      "status": "active или completed",
      "deliveryDate": "дата доставки",
      "address": "объект адреса",
      "orderPrice": "стоимость товаров",
      "deliveryPrice": "стоимость доставки",
      "totalPrice": "общая стоимость",
      "totalItems": "количество товаров",
      "items": "массив товаров в заказе"
    }
  ]
}
```

#### wallet_data.json
Содержит данные кошельков пользователей в формате:
```json
{
  "accounts": {
    "user_id": {
      "account_id": {
        "id": "идентификатор счета",
        "type": "card или savings",
        "balance": "баланс в рублях"
      }
    }
  },
  "transactions": {
    "user_id": [
      {
        "amount": "сумма транзакции (+ доход, - расход)",
        "title": "описание",
        "time": "время транзакции",
        "icon": "URL иконки (опционально)"
      }
    ]
  },
  "daily_topups": {
    "user_id": {
      "YYYY-MM-DD": "сумма пополнений за день"
    }
  },
  "user_phones": {
    "user_id": "номер телефона"
  }
}
```

#### blocked_tokens.json
Содержит массив заблокированных JWT токенов.

#### created_tokens.csv
Содержит список созданных JWT токенов для отслеживания.

### Загрузка данных

Данные загружаются автоматически при запуске приложения через config. Если файлы не найдены или содержат ошибки, приложение продолжит работу с пустыми данными и выведет предупреждение в логах.

### Расширение данных

Для добавления новых товаров или категорий просто отредактируйте соответствующие JSON файлы. Приложение автоматически подхватит изменения при следующем запуске.

### Восстановление из бэкапа

В приложении создаются бекапы данных каждые 24 часа. Для восстановления из бэкапа необходимо:

1. Скопировать файлы из `data/backups/YYYY-MM-DD/` в `data/`
2. Переименовать файлы бэкапа в стандартные имена:
   - `user_profiles_backup_*.json` → `user_profiles.json`
   - `cart_items_backup_*.json` → `cart_items.json`
   - И т.д.
3. Перезапустить приложение

