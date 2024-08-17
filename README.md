# coachwise API

## Database Query system
**Temp Note**: *would update db query related documents here*

## E2E Tests
```
go test -v ./tests -c test.config.yml
```

## Migration system
 **New migrations :**
 ```
 go run cmd/migrate/main.go new <migration_name>
 ```
 **Apply migrations :**
 ```
 go run cmd/migrate/main.go up
 ```


## Quick start
**should take care of matching config file to related connection such as pg and nats**
```
$ cd coachwise-api
$ cp .tmp.config.yml config.yml
$ sudo docker-compose up -d
$ go get
$ go run cmd/migrate/main.go up
$ go run cmd/app/main.go
``` 

