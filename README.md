# Отчет о проделанной работе

Для знакомства с MongoDb будем использовать [следующий датасет](https://www.kaggle.com/najzeko/steam-reviews-2021), посвященный отзывам пользователей Steam за 2021 год. Он достаточно большой, чтобы мы смогли ощутить всю мощь индексов.

## Запросы

### Запрос 1
В детстве моей любимой игрой была Counter-Strike Source. Давайте посмотрим, что о ней говорят в 2021 году:

`db.reviews2021.find({app_name: "Counter-Strike: Source", language: "russian"}, {app_name: 1, app_id: 1, review: 1, author: {steamid: 1, playtime_at_review: 1}}).skip(322).limit(10)`

![query1](https://user-images.githubusercontent.com/45429125/157709754-c36fc575-f168-48cf-a4b5-c093cbf3e97b.png)

Классика все еще актуальна, что не может не радовать.


### Запрос 2
Давайте поменяем отзыв:

`db.reviews2021.updateOne({_id: ObjectId("622a08a841b05a645301fb84")}, {$set: {review: "Любимая игра детства"}})`

![image](https://user-images.githubusercontent.com/45429125/157710358-e00739c3-0f49-47bc-8b99-02c69259e11d.png)


### Запрос 3
Как известно, Ведьмак 3, несмотря на то, что вышел более 5 лет назад и по сей день считается одной из лучших RPG за всю историю. Но есть ли несогласные с данным утверждением? Проверим:

`db.reviews2021.find({app_name: "The Witcher 3: Wild Hunt", language: "russian", review: {$regex: "отстой", $options: "i"}}, {app_name: 1, app_id: 1, review: 1, author: {steamid: 1, playtime_at_review: 1}}).sort({app_name: -1}).limi`

![query3](https://user-images.githubusercontent.com/45429125/157711076-79a44dae-e5af-42f1-9d7b-6eff67e1dd73.png)


### Запрос 4

Давайте избавимся от некоторых хейтеров!

`db.reviews2021.updateMany({app_name: "The Witcher 3: Wild Hunt", language: "russian", review: {$regex: "Лучшая RPG за всю историю!", $options: "i"}}, {$set: {review: "отстой"}})`

![query_4](https://user-images.githubusercontent.com/45429125/157711803-4c42b2cb-574b-468d-be66-3a0493074d69.png)


## Что там с индексами?

Создадим простенький индекс и посмотрим, что произошло со временем работы запросов 1 и 3.

`db.reviews2021.createIndex({app_name: 1})`

![index](https://user-images.githubusercontent.com/45429125/157713012-e58761e1-09b7-4f25-a48f-00c51e5cc710.png)



Проведем сравнения времени работы и числа просмотренных документов для первого запроса без индекса и с индексом (1 и 2 скриншоты соотв.):

`db.reviews2021.find({app_name: "Counter-Strike: Source", language: "russian"}, {app_name: 1, app_id: 1, review: 1, author: {steamid: 1, playtime_at_review: 1}}).skip(322).limit(10).explain("executionStats")`


![stats1](https://user-images.githubusercontent.com/45429125/157715335-38aa60e0-8200-4d4f-9a91-4f7072d1c5d3.png) ![stats3](https://user-images.githubusercontent.com/45429125/157713726-22e51eb4-bfd1-41ac-95c2-818bd5f48210.png)


Аналогично для запроса под номером 3: 

![stats2](https://user-images.githubusercontent.com/45429125/157713851-3b5ed8b2-a8e1-4737-9a73-f38b760dd3c0.png) ![stats3](https://user-images.githubusercontent.com/45429125/157713865-3547830f-c770-4737-a769-4ed7b6bab280.png)


## Вывод

Как можно заметить, удачно подобраный индекс достаточно сильно ускоряет время работы наших запросов. 
