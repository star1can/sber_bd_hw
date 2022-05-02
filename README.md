# HBASE

## Введение

Apache HBase – это нереляционная, распределенная база данных с открытым исходным кодом, написанная на языке Java по аналогии BigTable от Google. Изначально эта СУБД класса NoSQL создавалась компанией Powerset в 2007 году для обработки больших объёмов данных в рамках поисковой системы на естественном языке. Проектом верхнего уровня Apache Software Foundation HBase стала в 2010 году. На данный момент стабильной актуальной версией является 2.4.11.

СУБД относится к категории wide-column store и представляет собой колоночно-ориентированное, мультиверсионное хранилище типа key-value. Она работает поверх распределенной файловой системы HDFS и обеспечивает возможности BigTable для Hadoop, реализуя отказоустойчивый способ хранения больших объёмов разреженных данных. 


## Модель данных

Модель данных HBase реализуется по типу ключ-значение – <table, RowKey, Column Family, Column, timestamp> -> Value:

- данные организованы в таблицы, проиндексированные первичным ключом: RowKey;
- для каждого первичного ключа может храниться неограниченный набор атрибутов - колонок;
- колонки организованны в группы колонок (Column Family). Обычно в одну группу объединяют колонки с одинаковым шаблоном использования и хранения. Список и названия групп колонок фиксирован и имеет четкую схему. На уровне группы колонок задаются такие параметры как time to live (TTL) и максимальное количество хранимых версий;
- для каждого атрибута может храниться несколько различных версий. Разные версии имеют разный штамп времени (timestamp);
- записи физически хранятся в порядке, отсортированном по первичному ключу. При этом информация из разных колонок хранится отдельно, благодаря чему можно считывать данные только из нужного семейства колонок, тем самым, ускоряя операцию чтения;
- атрибуты, принадлежащие одной группе колонок и соответствующие одному ключу, физически хранятся как отсортированный список. Любой атрибут может отсутствовать или присутствовать для каждого ключа. Отсутствие атрибута не влечет никаких накладных расходов на хранение пустых значений;
- если разница между штампом времени (timestamp) для определенной версии и текущим временем больше TTL, такая запись помечается к удалению. Аналогично, если количество версий для определённого атрибута превысило максимальное количество версий.


### Пример

Рассмотрим как привычное многим представление переходит в колоночно-ориентированное:

![image](https://user-images.githubusercontent.com/45429125/166156233-970ecbd9-a171-4663-882c-a01035ee5190.png)


И обратно:

В данном случае имеется две ColumnFamily: data и meta. Первая содержит в себе колонку с именем "", а вторая - mimetype и size. Также у каждой записи в правом нижнем углу можно увидеть значения timestamp-ов.

![image](https://user-images.githubusercontent.com/45429125/166156243-db3b9965-7ead-4a3f-a38c-339b6251dada.png)

А так это будет выглядеть в виде, привычной многим, таблицы: (колонка Counters - служебная и не особо нам интересна на данном этапе)

![image](https://user-images.githubusercontent.com/45429125/166156686-d40c6adf-805d-479b-9316-4ab022a9f4ad.png)


## Архитектура

### Регионы

Распределение данных по разным физическим машинам кластера обеспечивается механизмом регионирования – автоматической горизонтальной группировки табличных строк.

Регион — это диапазон записей, соответствующих определенному диапазону подряд идущих первичных ключей. Каждый регион характеризуется некоторым количеством файлов hdfs и обслуживается Region Server, который в свою очередь является HDFS DataNode.

Вот так выглядит физическое представление Region Server:

![image](https://user-images.githubusercontent.com/45429125/166157443-131b529b-b226-4fdf-b71e-48368da1958a.png)

Как можно понять из картинки, регионирование - есть ничто иное, как автоматическое горизонтальное шардирование.



Каждый регион содержит следующие параметры:

- Data Storage - основное хранилище данных в Hbase. Данные физически хранятся на HDFS, в специальном формате - HFile, отсортированные по значению первичного ключа (RowKey). Одной паре (region, column family) соответствует как минимум один HFIle;
- MemStore - буфер для запись, специально выделенная область памяти для накопления данных перед записью, поскольку обновлять каждую запись в отсортированном HFile довольно дорого. Как только MemStore наполнится до некоторого критического значения, создается очередной HFile, куда будут записаны новые данные. Один на ColumnFamily;
- BlockCache - кэш на чтение;
- Write Ahead Log (WAL) – файл для логгирования всех операций с данными, чтобы их можно было восстановить в случае сбоя. Является единственным для каждого RegionServer;


Для наглядности:

![image](https://user-images.githubusercontent.com/45429125/166157114-0da8914f-abf7-44a7-aaa5-072f1dff9cf8.png)


### Zookeeper, master and slave

Введем следующие понятия:

- Master Server - главный узел в кластере, управляющий распределением регионов по региональным серверам, включая ведение их реестра, управление запусками регулярных задач и других организационных действий;
- ZooKeeper - распределенный сервис синхронизаций. Подробнее c данным "зверем" можно ознакомится самостоятельно.

Итак, наша модель взаимодействия выглядит следующим образом: (Store == Column Family)

![image](https://user-images.githubusercontent.com/45429125/166157282-4ad77014-0a36-4818-ae14-388abf48fad0.png)

#### Запись

Рассмотрим подробнее, как происходит запись:

1. Client хочет записать некоторые данные. Чтобы понять куда писать, клиент обращается к zookeeper и получает адрес Root Region;
2. Client обращается к Root region и получает адрес нужного RegionServer из таблицы META;
3. Client обращается к этому серверу и производит запись:
    - производится запись в HLOG;
    - далее, запись в MemStore. Как только MemStore наполнится до некоторого критического значения, создается очередной HFile, куда будут записаны новые данные.


#### Compaction

Изначально таблица состоит из одного региона, который разбивается на новые по мере роста (после превышения конкретно заданного порогового размера). Когда таблица становится слишком большой для одного сервера, она обслуживается кластером, на каждом узле которого размещается подмножество регионов таблицы. Таким образом, регионирование обеспечивает распределение нагрузки на таблицу. Совокупность отсортированных регионов, доступных по сети, образует общее содержимое распределенной таблицы:

![hbase_2](https://user-images.githubusercontent.com/45429125/166156934-7de5d18c-b586-453e-9823-0beac4346643.png)


Поскольку данные по одному региону могут храниться в нескольких HFile, для ускорения работы HBase периодически объединяет их, выполняя одну из 2-х операций под названием compaction:

- Minor Compaction, который запускается автоматически и выполняется в фоновом режиме. Имеет низкий приоритет по сравнению с другими операциями.
- Major Compaction, запускаемый вручную или в случае наступления определенных условий (триггеров), например, срабатывание по таймеру. Имеет высокий приоритет и может существенно замедлить работу кластера. Эту операцию рекомендуется выполнять при невысокой нагрузке на кластер. Во время выполнения Major Compaction также происходит физическое удаление данных, ранее помеченных соответствующей меткой tombstone.

#### Чтение

Как можно заметить, в текущей архитектуре нет единого места где хранится весь ряд(Row), ибо данные распределены по ColumnFamily, каждая из которых обладает своим MemStore, а после этого Region Server придется просмотреть конкретные Hfile-ы:

![8-zzwzcoed7obxmlgzxl3ihrzzw](https://user-images.githubusercontent.com/45429125/166234781-0bbf739a-0183-4898-aa9a-4818a7bf1ba8.png)


Прокомментируем:

1. Client хочет прочитать некоторую Raw(ряд) на Region Server;
2. Region Server смотрит в MemStore;
3. Если ничего не найдено, то смотрит уже в StoreFile и проверяет tombstone маркер
4. Кэширует операцию



## CRUD

Самый простой способ начать работу с Hbase — воспользоваться утилитой hbase shell. Она доступна сразу после установки hbase на любой ноде кластера hbase.
Как и большинство других hadoop-related проектов hbase реализован на языке java, поэтому и нативный api доступен для языке java. Native API довольно неплохо задокументирован на официальном сайте. 

Для работы из других языков программирования Hbase предоставляет Thrift API и Rest API. На базе них построены клиенты для всех основных языков программирования: python, PHP, Java Script и тд.

В нашем случае мы будем использовать shell.

Перед тем как продолжить повествование, развернем нашу базу данных в докере и запустим hbase shell:

![image](https://user-images.githubusercontent.com/45429125/166169125-96b9ae06-741b-4280-b93e-edfb07b313a7.png)

### Create

`create ‘<table name>’,’<column family>’`

При создании таблицы, также можно указать число версий для каждой Column Family, синтаксис следующий:

`create '<table name>', {NAME => 'column family1', VERSIONS => N}`


Создадим таблицу employees, в которой будет две Column Family: personal_data (versions=4) и professional_data (versions=5). Сделаем это: 

`create 'employees', {NAME => 'personal_data', VERSIONS => 4}, {NAME => 'professional_data', VERSIONS => 5}`


Проверим, что таблица действительно создалась, это можно сделать командой:

`list`

![image](https://user-images.githubusercontent.com/45429125/166173117-615386fe-4473-4e35-8a5c-33bf339c2a1a.png)


### Put

`put '<table name>','row1','<colfamily:colname>','<value>'`

Put – добавить новую или обновить существующую запись. Временной штамп (timestamp) этой записи может быть задан вручную или установлен автоматически как текущее время.
Добавим несколько записей в нашу таблицу:

`put 'employees','1','personal_data:name','Aleksandr'`

`put 'employees','1','personal_data:city','Moscow'`

`put 'employees','1','professional_data:position','manager'`

`put 'employees','1','professional_data:salary','100000'`


Вскоре Александр пошел на повышение: стал главным менеджером с зарплатой в 150 000 у.е и переехал в элитный район Воронежа,. Добавим соответствующие записи в таблицу. Важно, мы не хотим удалять прошлую информацию о нашем сотруднике: 

`put 'employees','1','personal_data:city','Voronezh'`

`put 'employees','1','professional_data:position','lead manager'`

`put 'employees','1','professional_data:salary','150000'`

Для просмотра наших данных мы воспользуемся командой, описанной ниже.


### Get

`get '<table name>','rowid', {COLUMN ⇒ 'column family:column name', VERSIONS => N}`

Get – получить данные по определенному первичному ключу (RowKey). Можно указать Column Family, откуда будет считана информация и количество версий, которые требуется прочитать. Посмотрим на первую строку нашей таблицы:

`get 'employees','1'`

![image](https://user-images.githubusercontent.com/45429125/166173677-bb2bfe8a-e17e-4d18-8b24-75e380c95f04.png)

А теперь посмотрим на все версии данных:

`get 'employees','1', {COLUMN => ['personal_data:name','personal_data:city','professional_data:position','professional_data:salary'] , VERSIONS => 5}`

![image](https://user-images.githubusercontent.com/45429125/166174170-2674f7ac-908b-4947-8378-81ad018d56cb.png)


### Scan

`scan '<table name>'`

Scan – поочередное чтение записей, начиная с указанной. Можно указать запись, до которой следует читать или количество записей, которые необходимо считать. Также в параметрах операции отмечается Column Family, откуда будет производиться чтение и максимальное количество версий для каждой записи.

Просканируем по полной:

`scan 'employees', {VERSIONS => 5}`

![image](https://user-images.githubusercontent.com/45429125/166175138-3908c94b-0164-49da-b69d-a65ac17c93d9.png)

Произведем сканирование колонки professional_data:salary и посмотрим историю зарплат работников за все время:

`scan 'employees', {COLUMN => 'professional_data:salary', VERSIONS => 2}`

![image](https://user-images.githubusercontent.com/45429125/166228755-29e32ad7-8e9e-46de-b292-cc61ef7c29be.png)


### Delete 

`delete '<table name>', '<row>', '<column name >', '<time stamp>'`
    
Delete – пометить определенную версию к удалению. Физического удаления при этом не произойдет, оно будет отложено до следующего Major Compaction.

Удалим информацию о том, что наш работник проживал в городе Москва и сразу это проверим командой `scan`:

`delete 'employees', '1', 'personal_data:city',1651453906823`

![image](https://user-images.githubusercontent.com/45429125/166175343-a34dd445-fc9d-42b7-92a7-b04930d77289.png)

Как можно заметить, теперь при чтении выдается только актуальное место проживания. 

## Индексы

Первичным индексом является rowkey, поддержка иных индексов на данный момент отсутствует.


## Транзакции

Apache HBase не поддерживает транзакции в общепринятом смысле, т. е. не дает возможности начать, а затем откатить операцию. Единственными транзакционными гарантиями, которые предоставляет HBase, являются:

- Атомарность операций на уровне строк;
- Любая операция сканирования, выполняемая в регионе HBase, увидит состояние данных, которое было в начале сканирования. Он не увидит новых данных, которые были записаны в регион во время сканирования.

Один из подходов решения этой проблемы заключается в использовании другого механизма поверх HBase, который обеспечивает семантику, подобную SQL. Apache Phoenix — один из таких проектов с очень активным сообществом разработчиков. 


## Back up

Для каждой из таблиц можно настроить репликацию, также есть поддержка snapshot-ов.


## Безопасность 

- В Hbase поддерживается механизм управления доступа на основе ролей (RBAC). С помощью него можно определить, какие пользователи или группы могут читать и записывать данный ресурс HBase.

- Метки видимости (VL) позволяют помечать ячейки и контролировать доступ к помеченным ячейкам, чтобы дополнительно ограничить, кто может читать или записывать определенные подмножества ваших данных. Метки видимости хранятся в виде тегов.

- Присутствует механизм прозрачного шифрования данных, не позволяющий злоумышленнику, который имеет доступ к файловой системе, изменять данные.


## Итоги:

- Apache HBase создавалась компанией Powerset в 2007 году для обработки больших объёмов данных в рамках поисковой системы на естественном языке. Проектом верхнего уровня Apache Software Foundation HBase стала в 2010 году;
- Основными инструментами взаимодействия являются: shell, Java Native API, Thrift API и Rest API;
- Hbase работает поверх распределенной файловой системы HDFS и обеспечивает возможности BigTable для Hadoop;
- Распределение данных по разным физическим машинам обеспечивается механизмом регионирования – автоматической горизонтальной группировки табличных строк;
- Hbase написана на языке Java;
- Первичным индексом является rowkey, поддержка вторичных индексов на данный момент отсутствует, но всегда можно сделать костыль в виде второй таблицы :)
- Понятие "план запроса" отсутствует;
- Не поддерживвает транзакции, но есть определенные гарантии на атомарность операций;
- Есть поддержка репликаций и снапшотов;
- В Hbase используется автоматический горизонтальный шардинг, обеспечиваемый механизмом регионирования;
- Есть поддержка механизма Map Reduce;
- Hbase, де-факто и де-юре, создана для Big Data и Data Mining, но для OLAP, OLPT и Data Warehousing лучше воспользоваться иными решениями, работающими поверх hbase (Hive, Phoenix, e.t.c);
- В качестве методов защиты используются следующие механизмы: RBAC, VL, шифрование данных в состоянии покоя;
- 

