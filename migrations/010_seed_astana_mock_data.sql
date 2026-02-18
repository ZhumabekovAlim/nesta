-- +goose Up
WITH city_upsert AS (
    INSERT INTO cities (id, name, is_active)
    VALUES ('astana', 'Астана', TRUE)
    ON CONFLICT (id) DO UPDATE
        SET name = EXCLUDED.name,
            is_active = EXCLUDED.is_active
    RETURNING id
),
subscription_type_upsert AS (
    INSERT INTO subscription_types (id, title, subtitle, price_cents, features, is_active)
    VALUES (
        'standard',
        'Стандарт',
        'Базовый тариф для ежедневного вывоза',
        9900,
        jsonb_build_array(
            'Вынос мусора 1 раз в день',
            'До 2 пакетов',
            'Выбор времени суток',
            'Техподдержка'
        ),
        TRUE
    )
    ON CONFLICT (id) DO UPDATE
        SET title = EXCLUDED.title,
            subtitle = EXCLUDED.subtitle,
            price_cents = EXCLUDED.price_cents,
            features = EXCLUDED.features,
            is_active = EXCLUDED.is_active
    RETURNING id
)
INSERT INTO residential_complexes (
    id,
    name,
    address,
    city,
    city_id,
    status,
    threshold_n,
    current_requests
)
SELECT
    v.id,
    v.name,
    v.address,
    'Астана',
    'astana',
    'approved',
    10,
    0
FROM (
    VALUES
        ('greenline', 'ЖК Greenline', 'проспект Туран, 55'),
        ('highvill-ishim', 'ЖК Highvill Ishim', 'улица Нажимеденова, 10/1'),
        ('highvill-gold-ishim', 'ЖК Highvill Gold Ishim', 'улица Нажимеденова, 4'),
        ('promenade-expo', 'ЖК Promenade Expo', 'улица Мангилик Ел, 52'),
        ('expo-boulevard', 'ЖК Expo Boulevard', 'улица Мангилик Ел, 56'),
        ('only-sun', 'ЖК Only Sun', 'проспект Улы Дала, 39'),
        ('only-soul', 'ЖК Only Soul', 'проспект Улы Дала, 41'),
        ('alpamys', 'ЖК Alpamys', 'улица Бухар Жырау, 30'),
        ('diplomat', 'ЖК Diplomat', 'улица Достык, 5'),
        ('diplomat-2', 'ЖК Diplomat 2', 'улица Достык, 5/1'),
        ('nursaya-1', 'ЖК Нурсая 1', 'улица Динмухамеда Кунаева, 12/1'),
        ('nursaya-2', 'ЖК Нурсая 2', 'улица Динмухамеда Кунаева, 12/2'),
        ('nursaya-3', 'ЖК Нурсая 3', 'улица Динмухамеда Кунаева, 12/3'),
        ('sezim-kala', 'ЖК Sezim Kala', 'улица Сыганак, 54А'),
        ('asylym', 'ЖК Asylym', 'улица Сауран, 18'),
        ('akbulak-town', 'ЖК Akbulak Town', 'улица Сарайшык, 5Е'),
        ('arena-city', 'ЖК Arena City', 'проспект Кабанбай Батыра, 59/1'),
        ('capital-park', 'ЖК Capital Park', 'проспект Мангилик Ел, 40'),
        ('capital-park-light', 'ЖК Capital Park Light', 'проспект Мангилик Ел, 42'),
        ('capital-park-flora', 'ЖК Capital Park Flora', 'проспект Мангилик Ел, 44'),
        ('park-avenue-ex', 'ЖК Park Avenue EX', 'улица Керей Жанибек хандар, 22'),
        ('park-avenue-exclusive', 'ЖК Park Avenue Exclusive', 'улица Керей Жанибек хандар, 24'),
        ('talan-towers-residences', 'ЖК Talan Towers Residences', 'улица Достык, 16'),
        ('emerald-towers', 'ЖК Emerald Towers', 'улица Туркестан, 20'),
        ('london', 'ЖК London', 'улица Алихан Бокейхана, 25'),
        ('apple-city', 'ЖК Apple City', 'улица Е-10, 17'),
        ('sensata-city', 'ЖК Sensata City', 'проспект Туран, 34/1'),
        ('gulder', 'ЖК Gulder', 'улица Айтматова, 45'),
        ('sat-city', 'ЖК Sat City', 'улица Е-312, 2'),
        ('sat-city-2', 'ЖК Sat City 2', 'улица Е-312, 4'),
        ('sat-city-3', 'ЖК Sat City 3', 'улица Е-312, 6'),
        ('inju-arena', 'ЖК Inju Arena', 'улица Е-10, 5'),
        ('nova-city', 'ЖК Nova City', 'проспект Туран, 57'),
        ('nova-city-south', 'ЖК Nova City South', 'проспект Туран, 59'),
        ('alpamys-2', 'ЖК Alpamys 2', 'улица Бухар Жырау, 32'),
        ('alanda', 'ЖК Alanda', 'улица Бейбитшилик, 14'),
        ('vremena-goda-winter', 'ЖК Времена года Зима', 'улица Кошкарбаева, 56'),
        ('vremena-goda-spring', 'ЖК Времена года Весна', 'улица Кошкарбаева, 58'),
        ('vremena-goda-summer', 'ЖК Времена года Лето', 'улица Кошкарбаева, 60'),
        ('vremena-goda-autumn', 'ЖК Времена года Осень', 'улица Кошкарбаева, 62'),
        ('shygys', 'ЖК Шыгыс', 'улица Кайыма Мухамедханова, 15'),
        ('zaman', 'ЖК Zaman', 'улица Айтеке би, 7'),
        ('barys-city', 'ЖК Barys City', 'проспект Туран, 48'),
        ('aq-didar', 'ЖК Aq Didar', 'улица Сауран, 3/1'),
        ('aq-bulak-river', 'ЖК Aq Bulak River', 'улица Кенесары, 70'),
        ('mirny-dom', 'ЖК Мирный дом', 'улица Иманова, 19'),
        ('jas-otan', 'ЖК Jas Otan', 'улица Рыскулбекова, 16'),
        ('dostar', 'ЖК Dostar', 'проспект Женис, 43'),
        ('ansar', 'ЖК Ansar', 'улица Сейфуллина, 47'),
        ('komfort-city-astana', 'ЖК Komfort City Astana', 'улица Туркестан, 14')
) AS v(id, name, address)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    address = EXCLUDED.address,
    city = EXCLUDED.city,
    city_id = EXCLUDED.city_id,
    status = EXCLUDED.status,
    threshold_n = EXCLUDED.threshold_n,
    current_requests = EXCLUDED.current_requests;

-- +goose Down
DELETE FROM residential_complexes
WHERE id IN (
    'greenline', 'highvill-ishim', 'highvill-gold-ishim', 'promenade-expo', 'expo-boulevard',
    'only-sun', 'only-soul', 'alpamys', 'diplomat', 'diplomat-2',
    'nursaya-1', 'nursaya-2', 'nursaya-3', 'sezim-kala', 'asylym',
    'akbulak-town', 'arena-city', 'capital-park', 'capital-park-light', 'capital-park-flora',
    'park-avenue-ex', 'park-avenue-exclusive', 'talan-towers-residences', 'emerald-towers', 'london',
    'apple-city', 'sensata-city', 'gulder', 'sat-city', 'sat-city-2',
    'sat-city-3', 'inju-arena', 'nova-city', 'nova-city-south', 'alpamys-2',
    'alanda', 'vremena-goda-winter', 'vremena-goda-spring', 'vremena-goda-summer', 'vremena-goda-autumn',
    'shygys', 'zaman', 'barys-city', 'aq-didar', 'aq-bulak-river',
    'mirny-dom', 'jas-otan', 'dostar', 'ansar', 'komfort-city-astana'
);

DELETE FROM subscription_types
WHERE id = 'standard';

DELETE FROM cities
WHERE id = 'astana';
