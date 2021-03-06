

# CHECKER

Предоставляет способ проверять и мониторить статус запущенных контейнеров. 

Для этого контейнеры должны предоставлять интерфейс для проверки, путем прописывания переменных окружения в Dockerfile содержащих инструкции для проверки.

Предоставляет три утилиты

* check - проверить запущенный контейнер и вернуть код возврата отличный от 0 если контейнер не работает.
* monitor - запустить мониторинг контейнера. Утилита будет периодически проверять контейнер и обновлять его статус в etcd.
* listener - утилита слушает ивенты докера и запускает мониторинг для всех запускаемых контейнеров.


## check

Проверяет контейнер и возвращает код возврата 0 если контейнер функционирует нормально.
Принимает на вход имя контейнера или его id.
Для проверки запускается команда описанная в переменной окружения HEALTHCHECK образа, которой передается ip адрес тестируемого контейнера.

Параметры запуска:

```
	check [options] container

	Options:
	-loglevel=  Уровень логгирования. Может быть DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL
	-stdout - булевский параметр включающий логгирование в stdout. По умолчанию ВКЛЮЧЕН.
	-syslog - булевский параметр включающий логгирование в stdout. По умолчанию ВЫКЛЮЧЕН.
```


## monitor
Запускает мониторинг контейнера. 
Периодически запускается утилита check и если контейнер функционирует нормально, мы прописываем в etcd ключ с информацией об этом контенте. 
Периодичность запуска определяется переменной окружения HEALTHCHECKPERIOD прописанной в Dockerfile проверяемого контейнера. Если такой переменной нет, 
периодичность задаяется равной 30 секундам.  

Если контейнер который мониторится удаляется из докера, то монитор завершает свою работу.


### Формат данных в etcd

Формат ключа в etcd описывающий сервис предоставляемый контейнером выглядит следующим образом.

/SERVICE/**IMAGENAME**/**CONTAINERID**

где **IMAGENAME** - последний сегмент имени образа. (Например для **tutum/rabbitmq** это будет **rabbitmq**, для **registry.at.netstream.ru/pusher/scheduler** это будет **scheduler** )
**CONTAINERID** - идентификатор контейнера.

Формат значения по этому ключу выглядит следующим образом:

Для случаев когда контейнер не пробрасывает порты наружу:

```
	{
		"IP": "172.17.28.214",
		"Name": "/server",
		"Ports": [
			{
				"IP": "",
				"PrivatePort": 0,
				"PublicPort": 9000,
				"Type": "tcp"
			}
		]
	}

```


Для случаев когда контейнер пробрасывает порты наружу:

```json
	{
		"IP": "172.17.0.3",
		"Name": "/server",
		"Ports": [
			{
				"IP": "10.10.14.35",
				"PrivatePort": 9000,
				"PublicPort": 49153,
				"Type": "tcp"
			}
		]
	}
```

TTL ключей равен HEALTHCHECKPERIOD * 2 + 1


### Параметры запуска

```
	Containers monitoring daemon.
	Started daemon which monitors container and updates status in etcd.
	Usage of monitor: monitor [options] container
	Options:
		-etcd-host="": Host where etcd is listenting
		-loglevel="INFO": Logging level. Must be one of (DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL)
		-stdout=false: Write logs to STDOUT. Default false
		-syslog=true: Write logs to SYSLOG. Default true


```

## listener

Listener это демон который мониторит события докера, и в случае старта нового контейнера 
запускается его мониторинг, который продолжается до тех пор, пока контейнер не будет остановлен или удален.

### Параметры запуска


```
	Docker event listener.
	Started daemon which monitors container and updates status in etcd.
	Usage of check: listener [options] container
	Options:
	-etcd-host="": Host where etcd is listenting
	-loglevel="INFO": Logging level. Must be one of (DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL)
	-stdout=false: Write logs to STDOUT. Default false
	-syslog=true: Write logs to SYSLOG. Default true

```


## Интерфейс который должны предоставлять проверяемые контейнеры

Для того чтобы контейнер можно было проверять посредством checker'а он должен удовлетворять следующим условиям:

* содержать внутри образа все инструменты которые требуются для проверки
* содержать внутри Dockerfile переменную окружения HEALTHCHECK содержащую команду которую нужно передать образу для запуска. 
	Эта команда должна принимать на вход ip адрес тестируемого контейнера
	и возвращать ненулевой код возврата, в случае если контейнер не функционирует правильно.
	Пример:
		HEALTHCHECK=ping
	такая команда будет пинговать тестируемый контейнер.
	Лучшим вариантом будет написать и положить внутрь контейнера скрипт для тестирования:
		HEALTHCHECK=/usr/bin/check.sh
* (опционально) содержать в Dockerfile переменную окружения HEALTHCHECKPERIOD в которой задается периодичность в секундах, 
	с которой должен проверяться этот контейнер.


При запуске команды для проверки ей передается ip адрес на котором запущен проверяемый 
