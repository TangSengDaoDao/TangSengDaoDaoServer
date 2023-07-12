build:
	docker build -t tangsengdaodaoserver .
push:
	docker tag tangsengdaodaoserver registry.cn-shanghai.aliyuncs.com/wukongim/tangsengdaodaoserver:latest-ultimate
	docker push registry.cn-shanghai.aliyuncs.com/wukongim/wukongchatserver:latest-ultimate
deploy:
	docker build -t tangsengdaodaoserver .
	docker tag tangsengdaodaoserver registry.cn-shanghai.aliyuncs.com/wukongim/tangsengdaodaoserver:latest-ultimate
	docker push registry.cn-shanghai.aliyuncs.com/wukongim/tangsengdaodaoserver:latest-ultimate
run-dev:
	docker-compose build;docker-compose up -d
stop-dev:
	docker-compose stop
env-test:
	docker-compose -f ./testenv/docker-compose.yaml up -d 